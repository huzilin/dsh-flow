// repo 包本文件：MySQL 错误码判定（唯一键冲突 / 死锁），供取号重试等场景消费。
package repo

import "github.com/go-sql-driver/mysql"

// MySQL 错误码。
const (
	// errCodeDuplicateEntry 唯一键冲突
	errCodeDuplicateEntry uint16 = 1062
	// errCodeDeadlock 死锁（含锁等待超时 1205 一并重试）
	errCodeDeadlock    uint16 = 1213
	errCodeLockWait    uint16 = 1205
	errCodeLockExpired uint16 = 3024
)

// IsDuplicateEntry 判定错误链路是否 MySQL 1062 唯一键冲突。
func IsDuplicateEntry(err error) bool {
	return matchMySQLErrorCode(err, errCodeDuplicateEntry)
}

// IsLockRace 判定错误链路是否锁竞争类（1213 死锁 / 1205 锁等待 / 3024 锁超时），
// 取号重试场景与唯一键冲突同等对待（整体重试）。
func IsLockRace(err error) bool {
	return matchMySQLErrorCode(err, errCodeDeadlock) ||
		matchMySQLErrorCode(err, errCodeLockWait) ||
		matchMySQLErrorCode(err, errCodeLockExpired)
}

// matchMySQLErrorCode 沿 Unwrap 与 pingcap Cause 双链找 *mysql.MySQLError 比错误码。
func matchMySQLErrorCode(err error, code uint16) bool {
	for cur := err; cur != nil; {
		if e, ok := cur.(*mysql.MySQLError); ok {
			return e.Number == code
		}
		if unwrapper, ok := cur.(interface{ Unwrap() error }); ok {
			cur = unwrapper.Unwrap()
			continue
		}
		causer, ok := cur.(interface{ Cause() error })
		if !ok {
			return false
		}
		cur = causer.Cause()
	}
	return false
}
