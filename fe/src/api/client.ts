import { ErrorResp } from "./proto/api";

/**
 * 统一 fetch client：protojson 编解码 + Bearer token 注入 + 统一失败出参解析
 * （口径 rd/handler/respond.go：成功 {ok:true,...}；失败 {ok:false,error,code,blocked?,missing?}，
 * HTTP 状态码由 errs.MapToHTTP 集中映射——401 UNAUTHORIZED / 409 OPTIMISTIC_LOCK·GATE_BLOCKED 等）。
 */

/** API 错误（任何非成功路径统一抛本类型）。 */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly blocked: boolean;
  readonly missing: string[];

  constructor(args: {
    status: number;
    code: string;
    message: string;
    blocked?: boolean;
    missing?: string[];
  }) {
    super(args.message);
    this.name = "ApiError";
    this.status = args.status;
    this.code = args.code;
    this.blocked = args.blocked ?? false;
    this.missing = args.missing ?? [];
  }

  /** 401：token 缺失/不匹配（fail closed 口径含后端未配置 token）。 */
  get isUnauthorized(): boolean {
    return this.status === 401;
  }

  /** 409 OPTIMISTIC_LOCK：乐观锁基线冲突（编辑表单须回传 GET 所得 updated_at 基线）。 */
  get isOptimisticLock(): boolean {
    return this.code === "OPTIMISTIC_LOCK";
  }

  /** 门禁拦截（blocked:true，如 deprecate 有引用 / delete 未 confirm）。 */
  get isBlocked(): boolean {
    return this.blocked;
  }
}

export interface ApiClientOptions {
  /** 同源部署留空（vite dev proxy /api）；跨域时填完整 origin */
  baseUrl?: string;
  /** Bearer token 取值函数（默认读 import.meta.env.VITE_WBFLOW_AUTH_TOKEN） */
  getToken?: () => string;
  /** fetch 注入点（测试用） */
  fetchImpl?: typeof fetch;
}

export interface ApiRequestArgs<Resp> {
  method: "GET" | "POST" | "PUT" | "DELETE";
  /** 以 / 开头的 API 路径（不含 query） */
  path: string;
  /** query 参数；值为 undefined/空串的键跳过 */
  query?: Record<string, string | number | boolean | undefined>;
  /** 请求体：调用方先经生成代码 toJSON 得 protojson 兼容对象 */
  body?: unknown;
  /** 响应解码：生成代码 fromJSON */
  decode: (raw: unknown) => Resp;
}

export class ApiClient {
  private readonly baseUrl: string;
  private readonly getToken: () => string;
  private readonly fetchImpl: typeof fetch;

  constructor(opts: ApiClientOptions = {}) {
    this.baseUrl = opts.baseUrl ?? "";
    this.getToken = opts.getToken ?? defaultGetToken;
    this.fetchImpl = opts.fetchImpl ?? fetch.bind(globalThis);
  }

  async request<Resp>(args: ApiRequestArgs<Resp>): Promise<Resp> {
    const headers: Record<string, string> = {};
    const token = this.getToken();
    // Bearer token 注入（空 token 不带头：后端 fail closed 返回 401）
    if (token !== "") {
      headers.Authorization = `Bearer ${token}`;
    }

    let body: string | undefined;
    if (args.body !== undefined) {
      headers["Content-Type"] = "application/json";
      body = JSON.stringify(args.body);
    }

    const res = await this.fetchImpl(`${this.baseUrl}${args.path}${buildQuery(args.query)}`, {
      method: args.method,
      headers,
      body,
    });

    const raw: unknown = await parseBody(res);
    if (!res.ok) {
      throw errorFromResp(res.status, raw);
    }
    // 2xx 但 ok:false（后端契约不应出现，防御性同形处理）
    if (isErrorRespShape(raw)) {
      throw errorFromResp(res.status, raw);
    }
    return args.decode(raw);
  }
}

/** 默认 client：同源 + env token。 */
export const api = new ApiClient();

function defaultGetToken(): string {
  return import.meta.env.VITE_WBFLOW_AUTH_TOKEN ?? "";
}

function buildQuery(query?: ApiRequestArgs<never>["query"]): string {
  if (!query) return "";
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    if (value === undefined || value === "") continue;
    params.set(key, String(value));
  }
  const s = params.toString();
  return s === "" ? "" : `?${s}`;
}

async function parseBody(res: Response): Promise<unknown> {
  const text = await res.text();
  if (text === "") return {};
  try {
    return JSON.parse(text);
  } catch {
    // 非 JSON 出参（如网关层纯文本）：保留状态码，消息退化为 statusText
    return { error: res.statusText || `HTTP ${res.status}` };
  }
}

function isErrorRespShape(raw: unknown): raw is Record<string, unknown> {
  return (
    typeof raw === "object" && raw !== null && "ok" in raw && (raw as { ok: unknown }).ok === false
  );
}

function errorFromResp(status: number, raw: unknown): ApiError {
  // 优先按 proto 契约 ErrorResp 解码（统一失败出参）
  try {
    const resp = ErrorResp.fromJSON(raw);
    return new ApiError({
      status,
      code: resp.code,
      message: resp.error,
      blocked: resp.blocked,
      missing: resp.missing,
    });
  } catch {
    // 解码失败兜底：原样文本
    const message =
      typeof raw === "object" && raw !== null && "error" in raw
        ? String((raw as { error: unknown }).error)
        : `HTTP ${status}`;
    return new ApiError({ status, code: "INTERNAL", message });
  }
}
