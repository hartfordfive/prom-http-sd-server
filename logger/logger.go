package logger

import (
	"go.uber.org/zap"
)

var Logger *zap.Logger

// func init() {
// 	Logger, err := zap.NewDevelopment() // or NewExample, NewProduction, or NewDevelopment
// 	if err != nil {
// 		fmt.Errorf("Could not initialize logger: %v", err)
// 		os.Exit(1)
// 	}
// 	defer Logger.Sync()

// }
