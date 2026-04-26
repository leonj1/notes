// Package sdk provides a strongly-typed Go client for the notes service
// REST API.
//
// # Installation
//
//	go get github.com/leonj1/notes/sdk@latest
//
// # Usage
//
//	import (
//	    "context"
//	    "log"
//	    "time"
//
//	    "github.com/leonj1/notes/sdk"
//	)
//
//	func main() {
//	    client := sdk.NewClient("http://notes.example.com",
//	        sdk.WithTimeout(10*time.Second),
//	    )
//
//	    schedule, err := client.CreateSchedule(context.Background(), sdk.Schedule{
//	        CronSchedule: "0 9 * * 1",
//	        ScriptPath:   "/scripts/report.sh",
//	        Description:  "Monday morning report",
//	        Status:       sdk.ScheduleStatusEnabled,
//	    })
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    log.Printf("created schedule %d", schedule.Id)
//	}
//
// # Versioning
//
// Releases are published as Git tags of the form sdk/vX.Y.Z, which the Go
// module system surfaces as semantic versions of github.com/leonj1/notes/sdk.
// The current package version is exposed as Version.
package sdk
