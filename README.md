# GORM ImmuDB Driver

## Quick Start

```go
import (
  immudb "github.com/0ctanium/gorm-immudb"
  "gorm.io/gorm"
)

// https://docs.immudb.io/master/develop/sqlstdlib.html
dsn := "immudb://immudb:immudb@127.0.0.1:3322/defaultdb"
db, err := gorm.Open(gimmudb.Open(dsn), &gorm.Config{})
```

## Configuration

```go
import (
  immudb "github.com/0ctanium/gorm-immudb"
  "gorm.io/gorm"
)

db, err := gorm.Open(immudb.New(immudb.Config{
  DSN: "immudb://immudb:immudb@127.0.0.1:3322/defaultdb", // data source name, refer https://docs.immudb.io/master/develop/sqlstdlib.html
  DefaultVarcharSize: 256, // add default size for string fields, not a primary key, no index defined and don't have default values
  DefaultBlobSize: 256, // add default size for bytes fields, not a primary key, no index defined and don't have default values
  DisableDeletion: true, // disable row deletion, which not supported before ImmuDB 1.2
}), &gorm.Config{})
```

## Customized Driver

```go
import (
  _ "example.com/my_immudb_driver"
  immudb "github.com/0ctanium/gorm-immudb"
  "gorm.io/gorm"
)

db, err := gorm.Open(immudb.New(immudb.Config{
  DriverName: "my_immudb_driver",
  DSN: "immudb://immudb:immudb@127.0.0.1:3322/defaultdb", // data source name, refer https://docs.immudb.io/master/develop/sqlstdlib.html
})
```

Checkout [https://gorm.io](https://gorm.io) for details.
