package playstore

import (
	"context"
	"io"
	"time"

	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

const (
	mediaHeader        = "application/octet-stream"
	chunkRetryDeadline = 60 * time.Second

	// https://developers.google.com/android-publisher/tracks
	TrackInternal   = "internal"
	TrackAlpha      = "alpha"
	TrackBeta       = "beta"
	TrackProduction = "production"

	// https://developers.google.com/android-publisher/api-ref/edits/tracks
	StatusCompleted   = "completed"
	StatusDraft       = "draft"
	StatusHalted      = "halted"
	StatusInProgress  = "inProgress"
	StatusUnspecified = "statusUnspecified"

	//https://developers.google.com/android-publisher/api-ref/rest/v3/edits.deobfuscationfiles#DeobfuscationFileType
	DeobfuscationFileTypeUnspecified = "deobfuscationFileTypeUnspecified"
	DeobfuscationFileProguard        = "proguard"
	DeobfuscationFile                = "nativeCode"
)

type IGService interface {
	IEditsService
	IUploadService
	IDraftService
}

type gService struct {
	*editsService
	*uploadService
	*draftService
}

func NewGEditsService(authFile string) (IGService, error) {
	edits, err := androidpublisher.NewService(context.Background(), option.WithCredentialsFile(authFile))
	if err != nil {
		return nil, err
	}
	return &gService{
		editsService:  &editsService{edits: edits.Edits},
		uploadService: &uploadService{edits: edits.Edits},
		draftService:  &draftService{edits: edits.Edits},
	}, nil
}

/**
 * Google API wrapper for edit creation, validation and commit
 */
type IEditsService interface {
	createEdit(packageName string) (string, error)
	validateEdit(packageName, editId string) error
	deleteEdit(packageName, editId string) error
	commitEdit(packageName, editId string) error
}

type editsService struct {
	edits *androidpublisher.EditsService
}

// createEdit creates an edit on playstore and returns editId
func (es *editsService) createEdit(packageName string) (string, error) {
	edit := &androidpublisher.AppEdit{}
	e, err := es.edits.Insert(packageName, edit).Do()
	if err != nil {
		return "", err
	}
	return e.Id, nil
}

// validateEdit validates edit for a given package on a playstore and returns error if edit validation failed
func (es *editsService) validateEdit(packageName, editId string) error {
	_, err := es.edits.Validate(packageName, editId).Do()
	return err
}

// deleteEdit deletes edit on playstore
func (es *editsService) deleteEdit(packageName, editId string) error {
	return es.edits.Delete(packageName, editId).Do()
}

// commits edit on playstore
func (es *editsService) commitEdit(packageName, editId string) error {
	_, err := es.edits.Commit(packageName, editId).Do()
	return err
}

/**
 * Google API wrapper for bundle and proguard mapping uploads
 */
type IUploadService interface {
	uploadBundle(r io.Reader, packageName, editId string) (appVersionCode int64, sha256 string, err error)
	uploadApk(r io.Reader, packageName, editId string) (appVersionCode int64, sha256 string, err error)
	uploadProguardMapping(r io.Reader, packageName, editId string, appVersionCode int64) error
}

type uploadService struct {
	edits *androidpublisher.EditsService
}

// uploadBundle uploads provided aab to playstore and returns upload version number and sha256 hash on success
func (us *uploadService) uploadBundle(r io.Reader, packageName, editId string) (appVersionCode int64, sha256 string, err error) {
	uRq := us.edits.Bundles.Upload(packageName, editId)
	uploaded, err := uRq.Media(r, googleapi.ContentType(mediaHeader), googleapi.ChunkRetryDeadline(chunkRetryDeadline)).Do()
	if err != nil {
		return -1, "", err
	}
	return uploaded.VersionCode, uploaded.Sha256, nil
}

// uploadApk uploads provided apk to playstore and returns upload version number and sha256 hash on success
func (us *uploadService) uploadApk(r io.Reader, packageName, editId string) (appVersionCode int64, sha256 string, err error) {
	uRq := us.edits.Apks.Upload(packageName, editId)
	uploaded, err := uRq.Media(r, googleapi.ContentType(mediaHeader), googleapi.ChunkRetryDeadline(chunkRetryDeadline)).Do()
	if err != nil {
		return -1, "", err
	}
	return uploaded.VersionCode, uploaded.Binary.Sha256, nil
}

// uploadProguardMapping uploads provided mappings file to playstore
func (us *uploadService) uploadProguardMapping(r io.Reader, packageName, editId string, appVersionCode int64) error {
	uRq := us.edits.Deobfuscationfiles.Upload(packageName, editId, appVersionCode, DeobfuscationFileProguard)
	_, err := uRq.Media(r, googleapi.ContentType(mediaHeader)).Do()
	return err
}

/**
 * Google API wrapper to create a simple draft
 */
type IDraftService interface {
	createDraft(packageName, editId, trackName string, appVersionCodes []int64) error
}

type draftService struct {
	edits *androidpublisher.EditsService
}

// createDraft creats a draft for given track and assigns appversions to it
func (ds *draftService) createDraft(packageName, editId, trackName string, appVersionCodes []int64) error {
	track := &androidpublisher.Track{
		Releases: []*androidpublisher.TrackRelease{
			{
				Status:       StatusDraft,
				VersionCodes: appVersionCodes,
			},
		},
		Track: trackName,
	}
	_, err := ds.edits.Tracks.Update(packageName, editId, trackName, track).Do()
	return err
}
