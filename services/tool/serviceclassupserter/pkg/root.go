package pkg

// This contains the command line code
import (
	"cloudservices/common/base"
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "serviceclassupserter",
		Short: "CLI for create/update/delete Service Classes",
	}
)

func init() {
	runUpserter := &cobra.Command{
		Use:   "run",
		Short: "Runs the upserter",
		Run:   runUpserter,
	}
	runUpserter.Flags().BoolVarP(&Cfg.DeleteOnMissing, "delete-missing", "m", Cfg.DeleteOnMissing, "Delete the Service Class if it exists only in the cloud. Default is false")
	runUpserter.Flags().BoolVarP(&Cfg.DisableDryRun, "disable-dry-run", "r", Cfg.DisableDryRun, "Enable/disable dry run. Default is false")
	runUpserter.Flags().StringVarP(&Cfg.DataDir, "data-dir", "d", Cfg.DataDir, "Data directory. Current working directory is the default")
	rootCmd.AddCommand(runUpserter)
}

func runUpserter(cmd *cobra.Command, args []string) {
	ctx := base.GetOperatorContext(context.Background())
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()
	fmt.Printf("Dry run is set to %t for data dir %s\n", !Cfg.DisableDryRun, Cfg.DataDir)
	err := UpsertServiceClasses(ctx, nil, CreateServiceClasses, UpdateServiceClasses, DeleteServiceClasses)
	if err != nil {
		fmt.Printf("Error occurred in the command. Error: %s\n", err.Error())
	} else {
		fmt.Printf("Command completed successfully\n")
	}
}

// Execute is the entry for the command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
