package playstore

import (
	"bytes"
	"crypto/rand"
	"io"
	"reflect"
	"testing"

	"github.com/spf13/afero"
)

func TestPublish(t *testing.T) {

	t.Run("should not allow empty packageName", func(t *testing.T) {
		// Arrange
		fs := afero.NewMemMapFs()
		authFile := "auth.json"
		binFile := "bin.aab"
		mockBins := Binaries(
			map[string]string{binFile: ""},
		)
		fs.Create(authFile)
		fs.Create(binFile)

		// Act
		_, err := Publish(fs, " ", TrackInternal, authFile, mockBins, false, false)

		// Assert
		if err == nil {
			t.Error("want error, got nil")
		}
	})

	t.Run("should verify auth file exists", func(t *testing.T) {
		// Arrange
		fs := afero.NewMemMapFs()
		appID := "com.sample.app0"
		authFile := "auth.json"
		binFile := "bin.aab"
		mockBins := Binaries(
			map[string]string{binFile: ""},
		)
		fs.Create(binFile)

		// Act
		_, err := Publish(fs, appID, TrackInternal, authFile, mockBins, false, false)

		// Assert
		if err == nil {
			t.Error("want error, got nil")
		}
	})

	t.Run("should verify all binary files exist", func(t *testing.T) {
		// Arrange
		fs := afero.NewMemMapFs()
		appID := "com.sample.app0"
		authFile := "auth.json"
		binFile := "bin.aab"
		mockBins := Binaries(
			map[string]string{binFile: "", "dontexit.aab": ""},
		)
		fs.Create(authFile)
		fs.Create(binFile)

		// Act
		_, err := Publish(fs, appID, TrackInternal, authFile, mockBins, false, false)

		// Assert
		if err == nil {
			t.Error("want error, got nil")
		}
	})

	t.Run("should verify all mapping exists", func(t *testing.T) {
		// Arrange
		fs := afero.NewMemMapFs()
		appID := "com.sample.app0"
		authFile := "auth.json"
		binFile := "bin.aab"
		mockBins := Binaries(map[string]string{binFile: "mapping.txt"})
		fs.Create(authFile)
		fs.Create(binFile)

		// Act
		_, err := Publish(fs, appID, TrackInternal, authFile, mockBins, false, false)

		// Assert
		if err == nil {
			t.Error("want error, got nil")
		}
	})

	t.Run("should not allow production track", func(t *testing.T) {
		// Arrange
		fs := afero.NewMemMapFs()
		appID := "com.sample.app0"
		authFile := "auth.json"
		binFile := "bin.aab"
		mockBins := Binaries(
			map[string]string{binFile: ""},
		)
		fs.Create(authFile)
		fs.Create(binFile)

		// Act
		_, err := Publish(fs, appID, TrackProduction, authFile, mockBins, false, false)

		// Assert
		if err == nil {
			t.Errorf("want error, got nil ")
		}
	})

	t.Run("given all files are present and valid, should return publish", func(t *testing.T) {
		// Arrange
		fs := afero.NewMemMapFs()
		appID := "com.sample.app"
		authFile := "auth.json"
		binFile := "bin.aab"
		fileMapping := "mapping.txt"
		mockBins := Binaries(
			map[string]string{binFile: fileMapping},
		)
		fs.Create(authFile)
		fs.Create(binFile)
		fs.Create(fileMapping)
		expected := publish{
			fs:          fs,
			packageName: appID,
			authFile:    authFile,
			files:       mockBins,
			track:       TrackInternal,
			apk:         false,
			verbose:     false,
		}

		// Act
		actual, err := Publish(fs, appID, TrackInternal, authFile, mockBins, false, false)

		// Assert
		if err != nil {
			t.Errorf("want no error, got: %v", err)
		}
		if actual == nil || !reflect.DeepEqual(*actual, expected) {
			t.Errorf("\nwant %+v\ngot %+v", expected, actual)
		}
	})
}

func TestUploadFiles(t *testing.T) {
	t.Run("Should call uploadBundle with expected binary file", func(t *testing.T) {
		// Arrange
		fs := afero.NewMemMapFs()
		fs.Create("auth.json")
		bin, actual, _ := createMockBinary(t, fs, "test.aab", "")
		publish, err := Publish(fs, "com.test.app", TrackInternal, "auth.json", []binary{bin}, false, false)
		if err != nil {
			t.Fatal(err)
		}
		gs := &mockGService{}

		// Act
		if err := publish.UploadFiles(gs); err != nil {
			t.Fatal(err)
		}

		// Assert
		if !bytes.Equal(actual, gs.bytes) {
			t.Errorf("want '%s' binary content, got '%s'", actual, gs.bytes)
		}
	})

	t.Run("Should call uploadBundle with expected binary file", func(t *testing.T) {
		// Arrange
		isApk := false
		fs := afero.NewMemMapFs()
		fs.Create("auth.json")
		bin, actual, _ := createMockBinary(t, fs, "test.aab", "")
		publish, _ := Publish(fs, "com.test.app", TrackInternal, "auth.json", []binary{bin}, isApk, false)

		gs := &mockGService{}

		// Act
		if err := publish.UploadFiles(gs); err != nil {
			t.Fatal(err)
		}

		// Assert
		if !bytes.Equal(actual, gs.bytes) {
			t.Errorf("want '%s' binary content, got '%s'", actual, gs.bytes)
		}
		if gs.uploadBundleCallCount != 1 {
			t.Errorf("want 1 call, got %d", gs.uploadBundleCallCount)
		}
	})

	t.Run("Should call uploadApk with expected binary file", func(t *testing.T) {
		// Arrange
		isApk := true
		fs := afero.NewMemMapFs()
		fs.Create("auth.json")
		bin, actual, _ := createMockBinary(t, fs, "test.apk", "")
		publish, _ := Publish(fs, "com.test.app", TrackInternal, "auth.json", []binary{bin}, isApk, false)
		gs := &mockGService{}

		// Act
		if err := publish.UploadFiles(gs); err != nil {
			t.Fatal(err)
		}

		// Assert
		if !bytes.Equal(actual, gs.bytes) {
			t.Errorf("want '%s' binary content, got '%s'", actual, gs.bytes)
		}
		if gs.uploadApkCallCount != 1 {
			t.Errorf("want 1 call, got %d", gs.uploadApkCallCount)
		}
	})

	t.Run("Should fail if sha256 on remote missmatch", func(t *testing.T) {
		// Arrange
		isApk := true
		fs := afero.NewMemMapFs()
		fs.Create("auth.json")
		bin, _, _ := createMockBinary(t, fs, "test.apk", "")
		publish, _ := Publish(fs, "com.test.app", TrackInternal, "auth.json", []binary{bin}, isApk, false)
		gs := &mockGService{}
		gs.Sha256 = "randomValue"

		// Act
		err := publish.UploadFiles(gs)

		//Assert
		if err == nil {
			t.Errorf("want error, got none")
		}
	})

	t.Run("Should call create and commit Edit", func(t *testing.T) {
		// Arrange
		isApk := true
		fs := afero.NewMemMapFs()
		fs.Create("auth.json")
		bin, _, _ := createMockBinary(t, fs, "test.apk", "")
		publish, _ := Publish(fs, "com.test.app", TrackInternal, "auth.json", []binary{bin}, isApk, false)
		gs := &mockGService{}

		// Act
		if err := publish.UploadFiles(gs); err != nil {
			t.Fatal(err)
		}

		//Assert
		if gs.commitEditCount != 1 {
			t.Errorf("want 1 commitEdit call, but got %d", gs.commitEditCount)
		}
		if gs.createEditCount != 1 {
			t.Errorf("want 1 createEdit call, but got %d", gs.createEditCount)
		}
		if gs.validateEditCount != 1 {
			t.Errorf("want 1 validateEdit call, but got %d", gs.validateEditCount)
		}
		if gs.deleteEditCount != 0 {
			t.Errorf("want no deleteEdit calls, bug got %d", gs.deleteEditCount)
		}
	})
}

// Helper mock service to seperate us from google libraries for testing
type mockGService struct {
	AppVersionCode        int64
	Sha256                string
	Error                 error
	packageName           string
	editId                string
	bytes                 []byte
	uploadBundleCallCount int64
	uploadApkCallCount    int64
	createEditCount       int64
	commitEditCount       int64
	validateEditCount     int64
	deleteEditCount       int64
}

func (gs *mockGService) uploadBundle(r io.Reader, packageName, editId string) (appVersionCode int64, sha256 string, err error) {
	gs.setFuncInputs(r, packageName, editId)
	gs.uploadBundleCallCount += 1
	return gs.AppVersionCode, gs.Sha256, gs.Error
}

func (gs *mockGService) uploadApk(r io.Reader, packageName, editId string) (appVersionCode int64, sha256 string, err error) {
	gs.setFuncInputs(r, packageName, editId)
	gs.uploadApkCallCount += 1
	return gs.AppVersionCode, gs.Sha256, gs.Error
}

func (gs *mockGService) uploadProguardMapping(r io.Reader, packageName, editId string, appVersionCode int64) error {
	gs.setFuncInputs(r, packageName, editId)
	gs.AppVersionCode = appVersionCode
	return gs.Error
}

func (gs *mockGService) setFuncInputs(r io.Reader, packageName, editId string) {
	b, _ := io.ReadAll(r)
	gs.bytes = b
	gs.packageName = packageName
	gs.editId = editId
	if gs.Sha256 == "" {
		rdr := r.(io.ReadSeeker)
		rdr.Seek(0, io.SeekStart)
		s, _ := fileSha256(rdr)
		gs.Sha256 = s
	}
}

func (gs *mockGService) createEdit(packageName string) (string, error) {
	gs.createEditCount += 1
	return "1", nil
}

func (gs *mockGService) validateEdit(packageName, editId string) error {
	gs.validateEditCount += 1
	return nil
}

func (gs *mockGService) deleteEdit(packageName, editId string) error {
	gs.deleteEditCount += 1
	return nil
}
func (gs *mockGService) commitEdit(packageName, editId string) error {
	gs.commitEditCount += 1
	return nil
}

func (gs *mockGService) createDraft(packageName, editId, trackName string, appVersionCodes []int64) error {
	return nil
}

func createMockBinary(t testing.TB, fs afero.Fs, binFile, mappingsFile string) (bin binary, binContent []byte, mappingsContent []byte) {
	t.Helper()
	if binFile == "" {
		return Binary(binFile), nil, nil
	}
	if mappingsFile == "" {
		b := createTestFile(t, fs, binFile, 10)
		return Binary(binFile), b, nil
	}
	b := createTestFile(t, fs, binFile, 10)
	b2 := createTestFile(t, fs, mappingsFile, 20)
	return BinaryWithMapping(binFile, mappingsFile), b, b2
}

func createTestFile(t testing.TB, fs afero.Fs, file string, size int64) []byte {
	t.Helper()

	f, err := fs.Create(file)
	if err != nil {
		t.Fatalf("failed creating '%s' test file: %s", file, err)
	}
	defer f.Close()

	r := io.LimitReader(rand.Reader, size)
	buf := make([]byte, size)
	_, err = r.Read(buf)
	if err != nil {
		t.Fatalf("failed reading data to buffer: %s", err)
	}

	_, err = f.Write(buf)
	if err != nil {
		t.Fatalf("failed writing data to file: %s", err)
	}
	return buf
}
