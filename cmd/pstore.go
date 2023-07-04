package cmd

import (
	"errors"
	"fmt"

	"github.com/sigitas-plk/playstore/playstore"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var (
	SecretFile string
	AppID      string
	AppBinOnly []string
	AppBin     map[string]string
	Track      string
	IsApk      bool
	Verbose    bool
)

var pstoreCmd = &cobra.Command{
	Use:   "pstore",
	Short: "Test CLI for appstore upload",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(AppBinOnly) == 0 && len(AppBin) == 0 {
			return errors.New("at leat one binary file to upload is required")
		}
		return upload()
	},
}

func init() {
	rootCmd.AddCommand(pstoreCmd)

	pstoreCmd.Flags().StringVar(&SecretFile, "authFile", "", "Authentication file")
	pstoreCmd.Flags().StringVar(&AppID, "appId", "", "Application ID e.g. com.sample.app")
	pstoreCmd.Flags().StringArrayVar(&AppBinOnly, "appBinOnly", []string{}, "Path to binary file to submit e.g. --appBinOnly my/app/path.aab")
	pstoreCmd.Flags().StringToStringVar(&AppBin, "appBin", map[string]string{}, "Key value pair with path to binary as key and its mappings as value. e.g. --appBin my/app/path.aab=may/mappings/mapth.txt")
	pstoreCmd.Flags().BoolVar(&IsApk, "apk", false, "Is apk (as opposed to app bundles .aab)")
	pstoreCmd.Flags().BoolVar(&Verbose, "verbose", false, "Verbose logging")

	pstoreCmd.MarkFlagRequired("authFile")
	pstoreCmd.MarkFlagRequired("appId")
}

func upload() error {

	if len(AppBinOnly) != 0 {
		for _, v := range AppBinOnly {
			// if same bin path set via --appBinOnly as with --appBin, use the one with the mappings
			if _, ok := AppBin[v]; !ok {
				AppBin[v] = ""
			}
		}
	}

	files := playstore.Binaries(AppBin)

	p, err := playstore.Publish(afero.NewOsFs(), AppID, playstore.TrackInternal, SecretFile, files, IsApk, Verbose)
	if err != nil {
		return fmt.Errorf("failed validating inputs: %w", err)
	}
	gs, err := playstore.NewGEditsService(SecretFile)
	if err != nil {
		return fmt.Errorf("failed creating new playstore service instance: %v", err)
	}
	if err := p.UploadFiles(gs); err != nil {
		return fmt.Errorf("failed uploading files: %v", err)
	}
	return nil
}
