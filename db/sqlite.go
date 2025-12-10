package db

import (
    "github.com/glebarez/sqlite"
    "github.com/rs/zerolog/log"
    "gorm.io/gorm"
)

const dbFileName = "/home/database/glkvm-cloud.db"

var deviceDB *gorm.DB

// GetDbClient returns the database client instance.
func GetDbClient() *gorm.DB {
    return deviceDB
}

// Init initializes the SQLite database connection and sets up logging.
func Init() {
    // Open a SQLite database connection
    db, err := gorm.Open(sqlite.Open(dbFileName), &gorm.Config{})
    if err != nil {
        log.Info().Msg(err.Error())
        // Panic if the database connection fails
        panic("failed to connect database")
    }
    // Set the global database client
    deviceDB = db

    // Auto-migrate the Device schema
    err = db.AutoMigrate(&DeviceMeta{})
    if err != nil {
        // Panic if auto-migration fails
        panic(err)
    }

    // Retrieve and log the initial data records
    list := make([]DeviceMeta, 0)
    db.Find(&list)
    log.Info().Msgf("==== SQLite init done ====, data record:%d \n", len(list))
}
