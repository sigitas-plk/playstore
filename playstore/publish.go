package playstore

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/mitchellh/ioprogress"
	"github.com/spf13/afero"
)

const (
	uploadProgressDrawInterval = 3 * time.Second
)

// binary aab or apk file and its mappings path
type binary struct {
	filePath    string
	mappingPath string
}

func BinaryWithMapping(path, mappingPath string) binary {
	return binary{
		filePath:    path,
		mappingPath: mappingPath,
	}
}

func Binary(path string) binary {
	return binary{
		filePath:    path,
		mappingPath: "",
	}
}

func Binaries(bins map[string]string) []binary {
	b := make([]binary, 0)
	for k, v := range bins {
		b = append(b, BinaryWithMapping(k, v))
	}
	return b
}

// func Binaries(b ...binary) []binary {
// 	return b
// }

// publish holds details on what we want to land on playstore
type publish struct {
	packageName string
	track       string
	authFile    string
	files       []binary
	apk         bool
	verbose     bool
	fs          afero.Fs
}

/**
 * Publish configuration of what should be uploaded
 *
 * fs - file system to enable easier testing
 * packageName - binary package name e.g. com.sample.app (you'll need at least one app submition)
 * track - which track this binary should be published to e.g. 'internal'
 * files - file(s) to be uploaded
 */
func Publish(fs afero.Fs, packageName, track, authFile string, files []binary, apk bool, verbose bool) (*publish, error) {

	p := &publish{
		verbose: verbose,
		fs:      fs,
	}

	if !p.fileExits(authFile) {
		return nil, fmt.Errorf("authentication file '%s' does not exist", authFile)
	}

	name := strings.TrimSpace(packageName)
	if name == "" {
		return nil, fmt.Errorf("package name must not be empty")
	}

	t := strings.TrimSpace(strings.ToLower(track))
	if t == "" {
		return nil, fmt.Errorf("track name to publish binary to is required")
	}
	if t != TrackBeta && t != TrackAlpha && t != TrackInternal {
		return nil, fmt.Errorf("provided track type '%s' not supported. Only supported types are '%s' '%s' '%s'", t, TrackBeta, TrackAlpha, TrackInternal)
	}

	if len(files) == 0 {
		return nil, errors.New("no files to upload provided")
	}

	for _, f := range files {
		if !p.fileExits(f.filePath) {
			return nil, fmt.Errorf("binary file '%s' does not exist", f.filePath)
		}
		if f.mappingPath != "" && !p.fileExits(f.mappingPath) {
			return nil, fmt.Errorf("mappings file '%s' does not exist", f.mappingPath)
		}
	}

	p.files = files
	p.authFile = authFile
	p.packageName = name
	p.track = t
	p.apk = apk
	return p, nil
}

/**
 * Meat and bones of this thing
 *
 * 1. creates an edit
 * 2. runs through list of files and uploads binaries + mappings if provided
 * 3. commits an edit
 */
func (p *publish) UploadFiles(gs IGService) error {

	if gs == nil {
		return errors.New("no Google Playstore service instance provided")
	}
	p.Debugf("starting file upload")
	edit, err := gs.createEdit(p.packageName)
	if err != nil {
		return err
	}
	p.Debugf("created edit on playstore with editId: %s", edit)

	versions := make([]int64, 0)
	for _, f := range p.files {

		v, err := p.upload(gs, f.filePath, edit, p.apk)
		if err != nil {
			gs.deleteEdit(p.packageName, edit)
			return err
		}
		versions = append(versions, v)
		if f.mappingPath == "" {
			p.Debugf("No mappings provided, skipping mapping upload for this file.")
			continue
		}

		if err := p.uploadMapping(gs, f.mappingPath, edit, v); err != nil {
			gs.deleteEdit(p.packageName, edit)
			return err
		}
	}

	p.Debugf("validating app submittion")
	if err := gs.validateEdit(p.packageName, edit); err != nil {
		gs.deleteEdit(p.packageName, edit)
		return err
	}

	if err := gs.commitEdit(p.packageName, edit); err != nil {
		gs.deleteEdit(p.packageName, edit)
		return err
	}

	log.Println("All files uploaded successfully.")
	return nil
}

func (p *publish) upload(us IUploadService, filePath, editId string, isApk bool) (version int64, err error) {

	p.Debugf("uploading %s", filePath)

	f, err := p.fs.Open(filePath)
	if err != nil {
		return -1, err
	}
	defer f.Close()

	hash, err := fileSha256(f)
	if err != nil {
		return -1, fmt.Errorf("failed calculating '%s' sha256 hash: %w", filePath, err)
	}

	pReader := &ioprogress.Reader{
		Reader:       f,
		Size:         p.fileSize(filePath),
		DrawFunc:     ioprogress.DrawTerminalf(log.Writer(), ioprogress.DrawTextFormatBytes),
		DrawInterval: uploadProgressDrawInterval,
	}

	uplF := us.uploadBundle
	if isApk {
		uplF = us.uploadApk
	}

	v, sha256, err := uplF(pReader, p.packageName, editId)
	if err != nil {
		return -1, err
	}
	p.Debugf("File successfully uploaded with appVersion: '%d'. Verifying file integrity on playstore", v)
	if sha256 != hash {
		return -1, fmt.Errorf("failed integrity verification with local file hash '%s' and remote '%s'", hash, sha256)
	}
	p.Debugf("File integrity check passed wtih sha256 '%s'", sha256)
	return v, nil
}

func (p *publish) uploadMapping(us IUploadService, filePath, editId string, appVersionCode int64) error {

	p.Debugf("Uploading mappgins '%s' for upload with appVersionCode '%d'", filePath, appVersionCode)

	f, err := p.fs.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	err = us.uploadProguardMapping(f, p.packageName, editId, appVersionCode)

	if err != nil {
		return err
	}
	p.Debugf("Mapping '%s' successfully uploaded.", filePath)
	return nil
}
