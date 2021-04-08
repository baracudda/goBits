package sqlBits

import (
	"database/sql"
	"database/sql/driver"
	"reflect"
)

type DriverName string

const (
	MySQL DriverName = "MySQL"
	PostgreSQL DriverName = "PostgreSQL"
	SQLite DriverName = "SQLite3"
)

type DriverInfo struct {
	// The database/sql API doesn't provide a way to get the registry name for
	// a driver from the driver type.
	Name DriverName
	Type reflect.Type
	// The rune used around table/field names in case of spaces and keyword clashes.
	// Determined by the database type being used (MySQL vs Oracle, etc.).
	IdentifierDelimiter rune
	// Not all drivers support named parameters; otherwise restricted to "$1" or "?".
	SupportsNamedParams bool
}

var DriverMeta map[reflect.Type]*DriverInfo

func (d *DriverInfo) SetDriverName( driverName string ) *DriverInfo {
	d.Name = DriverName(driverName)
	switch d.Name {
	case MySQL:
		d.IdentifierDelimiter = '`'
	case PostgreSQL:
		d.IdentifierDelimiter = '"'
	case SQLite:
		d.IdentifierDelimiter = '"'
	}
	return d
}

func RegisterDriverInfo( driverName string, dbDriver interface{} ) {
	driverType := reflect.TypeOf(dbDriver)
	DriverMeta[driverType] = (&DriverInfo{Type: driverType}).SetDriverName(driverName)
}

func init() {
	DriverMeta = map[reflect.Type]*DriverInfo{}
	for _, driverName := range sql.Drivers() {
		// Tested empty string DSN with MySQL, PostgreSQL, and SQLite3 drivers.
		db, _ := sql.Open(driverName, "")
		if db != nil {
			RegisterDriverInfo(driverName, db.Driver())
		}
	}
}

// SqlDriverToDriverName
// THANKS TO rbranson: https://github.com/golang/go/issues/12600#issuecomment-378363201
// The database/sql API doesn't provide a way to get the registry name for
// a driver from the driver type.
func SqlDriverToDriverName(driver driver.Driver) DriverName {
	driverType := reflect.TypeOf(driver)
	if driverInfo, found := DriverMeta[driverType]; found {
		return driverInfo.Name
	}
	return ""
}

func GetDriverMeta(dbDriver interface{}) *DriverInfo {
	driverType := reflect.TypeOf(dbDriver)
	if _, found := DriverMeta[driverType]; found {
		return DriverMeta[driverType]
	}
	return nil
}

type DbMetatater interface {
	GetDbMeta() *DriverInfo
}

type DbTransactioner interface {
	InTransaction() bool
	BeginTransaction()
	CommitTransaction()
	RollbackTransaction()
}
