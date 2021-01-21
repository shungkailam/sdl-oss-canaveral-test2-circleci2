package releases_test

import (
	"cloudservices/common/base"
	"cloudservices/operator/config"
	"cloudservices/operator/generated/operator/restapi/operations/edge"
	"cloudservices/operator/releases"
	"fmt"
	"github.com/stretchr/testify/require"
	"os"
	"os/exec"
	"strings"
	"time"

	"testing"

	"github.com/go-openapi/runtime"
)

func createReleaseTarFile(t *testing.T, releaseLocalDir, tarFilename, data string) {
	// figure how to upload files by using runtime file
	tempFile := strings.TrimSuffix(tarFilename, ".tgz")
	file, err := os.OpenFile(fmt.Sprintf("%s/%s", releaseLocalDir, tempFile), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	require.NoError(t, err)

	_, err = file.Write([]byte(data))
	if err != nil {
		file.Close()
		t.Fatal(err)
	}
	file.Close()
	cmd := exec.Command("sh", "-c", fmt.Sprintf("tar -C %s -czvf %s/%s.tgz %s", releaseLocalDir, releaseLocalDir, tempFile, tempFile))
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
}

func TestEdgeUpgrade(t *testing.T) {
	now := time.Now().UTC()
	*config.Cfg.S3Bucket = "sherlock-dev-releases"
	*config.Cfg.S3Prefix = fmt.Sprintf("operator-test%d/", now.UnixNano())
	*config.Cfg.S3Region = "us-west-2"

	releaseLocalDir := fmt.Sprintf("/tmp/%s", base.GetUUID())
	releaseFilename := "testfile.tgz"
	releaseFilepath := fmt.Sprintf("%s/%s", releaseLocalDir, releaseFilename)

	t.Logf("creating working folder %s", releaseLocalDir)
	err := os.MkdirAll(releaseLocalDir, 0777)
	require.NoError(t, err)

	// Clean up
	defer os.RemoveAll(releaseLocalDir)
	changelogData := `{
		"NewFeatures": {
		"PlatformServices": "test",
		"KubernetesUpgrade": "test",
		"OSUpgrade": "test"
		},
		"BugFixes": "test"
		}`
	upgradeData := "1st change"
	upgradeType := "minor"
	upgradeFiles := runtime.File{}

	createReleaseTarFile(t, releaseLocalDir, releaseFilename, upgradeData)

	changelogData2 := `{
		"NewFeatures": {
		"PlatformServices": "test2",
		"KubernetesUpgrade": "test2",
		"OSUpgrade": "test2"
		},
		"BugFixes": "test2"
		}`
	upgradeData2 := "2nd change"
	upgradeFiles2 := runtime.File{}

	releaseList, err := releases.GetLatestRelease()
	require.NoErrorf(t, err, "failed to get latest release %s", err)
	relTest1 := releaseList[0]

	// Read to get reader
	file, err := os.OpenFile(releaseFilepath, os.O_RDWR, 0666)
	require.NoError(t, err)

	upgradeFiles.Data = file
	ip := edge.UploadReleaseParams{Changelog: &changelogData, UpgradeType: upgradeType, UpgradeFiles: &upgradeFiles}
	opKey, err := releases.UploadRelease(ip)
	require.NoErrorf(t, err, "failed to upload file %s", err)
	file.Close()
	t.Logf(opKey)
	rel, err := releases.GetRelease(opKey)
	require.NoErrorf(t, err, "failed to get release %s: %s", opKey, err)
	if string(rel.Data) != "" || rel.Changelog != changelogData {
		t.Fatalf("upload data mismatch got %s %s wanted %s %s", string(rel.Data), string(rel.Changelog), upgradeData, changelogData)
	}
	t.Logf("Upload and get success")

	createReleaseTarFile(t, releaseLocalDir, releaseFilename, upgradeData2)

	file, err = os.OpenFile(releaseFilepath, os.O_RDWR, 0666)
	require.NoError(t, err)

	upgradeFiles2.Data = file

	t.Logf("Testing update")
	ip2 := edge.UpdateReleaseParams{ReleaseID: opKey, Changelog: &changelogData2, UpgradeFiles: &upgradeFiles2}
	opKey, err = releases.UpdateRelease(ip2)
	require.NoErrorf(t, err, "failed to upload file %s", err)
	file.Close()
	rel, err = releases.GetRelease(opKey)
	require.NoErrorf(t, err, "failed to get release %s: %s", opKey, err)
	if string(rel.Data) != "" || rel.Changelog != changelogData2 {
		t.Fatalf("upload data mismatch got %s %s wanted %s %s", string(rel.Data), string(rel.Changelog), upgradeData2, changelogData2)
	}

	t.Logf("Update and get success")

	// Test delete
	t.Logf("Testing delete")

	ip3 := edge.DeleteReleaseParams{ReleaseID: opKey}
	_, err = releases.DeleteRelease(ip3)
	require.NoErrorf(t, err, "failed to delete file %s", err)

	_, err = releases.GetAllReleases()
	require.NoErrorf(t, err, "failed to get all releases %s", err)

	releaseList, err = releases.GetLatestRelease()
	require.NoErrorf(t, err, "failed to get latest release %s", err)
	relTest2 := releaseList[0]

	if relTest1.ID != relTest2.ID {
		t.Fatalf("release does not match expected %s as %s", relTest1, relTest2)
	}
	t.Logf("Delete success")

}
