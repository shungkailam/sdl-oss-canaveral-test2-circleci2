package api_test

import (
	"archive/tar"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/operator/api"
	"cloudservices/operator/config"
	"cloudservices/operator/generated/operator/client/edge"
	"cloudservices/operator/generated/operator/client/operator"
	"cloudservices/operator/generated/operator/models"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"
	"time"

	httptransport "github.com/go-openapi/runtime/client"

	"cloudservices/operator/generated/operator/client"
	"os"

	"testing"

	gapi "cloudservices/operator/generated/grpc"

	"github.com/golang/glog"
)

func init() {
	apitesthelper.StartServices(&apitesthelper.StartServicesConfig{StartPort: 8190})
}

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

func extractSingleFile(t *testing.T, reader io.Reader) io.Reader {
	gzipReader, err := gzip.NewReader(reader)
	require.NoError(t, err)
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		if header == nil {
			continue
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		tokens := strings.Split(header.Name, "/")
		if strings.HasPrefix(tokens[len(tokens)-1], "._") {
			// Apple qurantine report
			continue
		}
		break
	}
	return tarReader
}

func TestUpgradeAPI(t *testing.T) {
	time.Sleep(5 * time.Second)
	now := time.Now().UTC()
	S3Bucket := "sherlock-dev-releases"
	S3Prefix := fmt.Sprintf("operator-test-bug%d/", now.UnixNano())
	S3Region := "us-west-2"
	cfg := client.TransportConfig{
		Host:     fmt.Sprintf("localhost:%d", *config.Cfg.Port),
		BasePath: client.DefaultBasePath,
		Schemes:  []string{"http"},
	}
	glog.Infof("Connect to operator: %#v", cfg)
	//basicAuth := httptransport.BasicAuth("xi-iot", "Z=rxn7s!efW[kevCV3+&")

	// Initilize the grpc and rest client
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	operatorClient := client.NewHTTPClientWithConfig(nil, &cfg)
	loginParams := &operator.LoginParams{Context: ctx}
	loginParams.LoginParams = &models.LoginParams{Email: "operator@ntnxsherlock.com", Password: "operator"}

	resp, err := operatorClient.Operator.Login(loginParams)
	require.NoError(t, err, "Login Failed")

	bearerAuth := httptransport.BearerToken(resp.Payload.Token)

	apiServer := api.NewAPIServer()
	tt := []struct {
		testName      string
		testChangelog string
		testFileName  string
		testData      string
	}{
		{
			testName: "Basic upgrade upload 1",
			testChangelog: `{
				"NewFeatures": {
				"PlatformServices": "test",
				"KubernetesUpgrade": "test",
				"OSUpgrade": "test"
				},
				"BugFixes": "test"
				}`,
			testFileName: "testfile1.tgz",
			testData:     "first test file data",
		},
		{
			testName: "Basic upgrade upload 2",
			testChangelog: `{
				"NewFeatures": {
				"PlatformServices": "test2",
				"KubernetesUpgrade": "test2",
				"OSUpgrade": "test2"
				},
				"BugFixes": "test2"
				}`,
			testFileName: "testfile1.tgz",
			testData:     "second test file data",
		},
	}

	for _, tc := range tt {
		*config.Cfg.S3Bucket = S3Bucket
		*config.Cfg.S3Prefix = S3Prefix
		*config.Cfg.S3Region = S3Region
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		var uploadedRelease string
		t.Run(fmt.Sprintf("Run upload upgrade %s", tc.testName), func(t *testing.T) {
			releaseLocalDir := fmt.Sprintf("/tmp/%s", base.GetUUID())
			releaseFilepath := fmt.Sprintf("%s/%s", releaseLocalDir, tc.testFileName)
			err := os.MkdirAll(releaseLocalDir, 0777)
			require.NoError(t, err)
			t.Logf("Using temp directory %s to create %s", releaseLocalDir, releaseFilepath)
			defer os.RemoveAll(releaseLocalDir)
			// Create a file and close it for upgrade
			createReleaseTarFile(t, releaseLocalDir, tc.testFileName, tc.testData)
			file, err := os.OpenFile(releaseFilepath, os.O_RDONLY, 0666)
			require.NoError(t, err)
			uploadReleaseInput := edge.UploadReleaseParams{Context: ctx, Changelog: &tc.testChangelog, UpgradeType: "major", UpgradeFiles: file}
			uploadReleaseOutput, err := operatorClient.Edge.UploadRelease(&uploadReleaseInput, bearerAuth)
			if err != nil {
				file.Close()
				t.Fatal("Failed to upload releases, Error:", err)
			}
			file.Close()
			t.Logf("Uploaded release %s successfully", uploadReleaseOutput.Payload)
			uploadedRelease = uploadReleaseOutput.Payload
		})
		t.Run(fmt.Sprintf("Check if we can get the upgrade using rest and grpc %s", tc.testName), func(t *testing.T) {

			listCompatibleReleasesInput := edge.ListCompatibleReleasesParams{Context: ctx, ReleaseID: "v0.0.0"}
			listCompatibleReleasesOutput, err := operatorClient.Edge.ListCompatibleReleases(&listCompatibleReleasesInput, bearerAuth)
			require.NoError(t, err, "Failed to list compatible releases, Error:", err)
			// listReleasesInput := edge.ListReleasesParams{Context: ctx}
			// listReleasesOutput, err := operatorClient.Edge.ListReleases(&listReleasesInput)
			// if err != nil {
			// 	log.Fatal("Failed to list releases, Error:", err)
			// }

			glistCompatibleReleasesInput := gapi.ListCompatibleReleasesRequest{Id: "v0.0.0"}
			glistCompatibleReleasesOutput, err := apiServer.RPCServer.ListCompatibleReleases(ctx, &glistCompatibleReleasesInput)
			require.NoError(t, err, "Failed to list compatible releases using grpc")

			glistReleasesInput := gapi.ListReleasesRequest{}
			glistReleasesOutput, err := apiServer.RPCServer.ListReleases(ctx, &glistReleasesInput)
			require.NoError(t, err, "Failed to list releases using grpc")

			if len(listCompatibleReleasesOutput.Payload) == 0 {
				t.Fatal("Failed to get uploaded release, Error:", err)
			}
			for idx := range listCompatibleReleasesOutput.Payload {
				if !(len(listCompatibleReleasesOutput.Payload) == len(glistCompatibleReleasesOutput.Releases) &&
					len(listCompatibleReleasesOutput.Payload) == len(glistReleasesOutput.Releases)) {

					t.Fatal("length of releases do not match, Error:", err)
				}
				if !(listCompatibleReleasesOutput.Payload[idx].ID == glistCompatibleReleasesOutput.Releases[idx].Id &&
					listCompatibleReleasesOutput.Payload[idx].ID == glistReleasesOutput.Releases[idx].Id) {
					t.Fatal("ID's of releases do not match, Error:", err)
				}
				if !(listCompatibleReleasesOutput.Payload[idx].Changelog == glistCompatibleReleasesOutput.Releases[idx].Changelog &&
					listCompatibleReleasesOutput.Payload[idx].Changelog == glistReleasesOutput.Releases[idx].Changelog) {
					t.Fatal("Changelog's of releases do not match, Error:", err)
				}
			}
		})

		t.Run(fmt.Sprintf("Check if uploaded data matches %s", tc.testName), func(t *testing.T) {
			gGetReleaseInput := gapi.GetReleaseRequest{Id: uploadedRelease}
			gGetReleaseOutput, err := apiServer.RPCServer.GetRelease(ctx, &gGetReleaseInput)
			require.NoError(t, err, "Failed to let release using grpc, Error:", err)
			resp, err := http.Get(gGetReleaseOutput.Url)
			require.NoError(t, err)
			defer resp.Body.Close()
			reader := extractSingleFile(t, resp.Body)
			data, err := ioutil.ReadAll(reader)
			require.NoError(t, err)
			if tc.testData != string(data) {
				t.Fatal("Release data does not match with what is expected")
			}

		})

		t.Run(fmt.Sprintf("Check if we get the correct output for various edge versions %s", tc.testName), func(t *testing.T) {
			tt2 := []struct {
				releaseID string
				result    string
				resultlen int
			}{
				{releaseID: "v0.0.0", result: uploadedRelease, resultlen: 1},
				{releaseID: uploadedRelease, result: "", resultlen: 0},
				{releaseID: "v2000.0.0", result: "", resultlen: 0},
			}
			for testID, tc2 := range tt2 {
				listCompatibleReleasesInput := edge.ListCompatibleReleasesParams{Context: ctx, ReleaseID: tc2.releaseID}
				listCompatibleReleasesOutput, err := operatorClient.Edge.ListCompatibleReleases(&listCompatibleReleasesInput, bearerAuth)
				require.NoError(t, err, "Failed to list compatible releases, Error:", err)
				// listReleasesInput := edge.ListReleasesParams{Context: ctx}
				// listReleasesOutput, err := operatorClient.Edge.ListReleases(&listReleasesInput)
				// if err != nil {
				// 	log.Fatal("Failed to list releases, Error:", err)
				// }

				glistCompatibleReleasesInput := gapi.ListCompatibleReleasesRequest{Id: tc2.releaseID}
				glistCompatibleReleasesOutput, err := apiServer.RPCServer.ListCompatibleReleases(ctx, &glistCompatibleReleasesInput)
				require.NoError(t, err, "Failed to list compatible releases using grpc")

				if len(listCompatibleReleasesOutput.Payload) != tc2.resultlen || len(glistCompatibleReleasesOutput.Releases) != tc2.resultlen {
					t.Fatalf("subtest %d Error:Release length does not match, got %d and %d expected %d,", testID, len(listCompatibleReleasesOutput.Payload),
						len(glistCompatibleReleasesOutput.Releases), tc2.resultlen)
				}

				for idx := range listCompatibleReleasesOutput.Payload {
					if !(listCompatibleReleasesOutput.Payload[idx].ID == uploadedRelease && uploadedRelease == glistCompatibleReleasesOutput.Releases[idx].Id) {
						t.Fatal("ID's of releases do not match, Error:", err)
					}
				}
			}

		})
		deleteReleaseInput := edge.DeleteReleaseParams{Context: ctx, ReleaseID: uploadedRelease}
		_, err := operatorClient.Edge.DeleteRelease(&deleteReleaseInput, bearerAuth)
		require.NoError(t, err, "Failed to delete release")
	}
}
