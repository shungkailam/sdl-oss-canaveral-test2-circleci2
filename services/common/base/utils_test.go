package base_test

import (
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/docker/distribution/uuid"
	"github.com/stretchr/testify/require"
)

type TestRangeString struct {
	ID string `validate:"range=1:200"`
}
type Test1 struct {
	TestRangeString
}

type TestEmail struct {
	Email string `validate:"email,range=5:100"`
}
type Test2 struct {
	TestEmail
}

type TestRangeInt struct {
	I int `validate:"range=3:100"`
}
type Test3 struct {
	TestRangeInt
}

type TestSliceRange struct {
	ID *string `validate:"range=2:200"`
}

type Test4 struct {
	Slices []TestSliceRange `validate:"range=1"`
}

type Test5 struct {
	Type string `validate:"options=AWS:GCP:AZURE,ignore=op1"`
}

func TestModelValidator(t *testing.T) {
	t.Run("Model validator test", func(t *testing.T) {
		t.Log("running model validator test")
		// String range test
		test1 := &Test1{TestRangeString: TestRangeString{ID: "h"}}
		err := base.ValidateStruct("test", test1, "")
		require.NoError(t, err)
		t.Logf("Passed for %+v", test1)

		// String range negative test
		test1 = &Test1{TestRangeString: TestRangeString{}}
		err = base.ValidateStruct("test", test1, "")
		require.Errorf(t, err, "Failed for %+v", test1)
		t.Logf("Error: %s", err.Error())

		// Email test
		test2 := &Test2{TestEmail: TestEmail{Email: "test@ntnxsherlock.com"}}
		err = base.ValidateStruct("test", test2, "")
		require.NoError(t, err)
		t.Logf("Passed for %+v", test2)

		// Email whitespace test
		test2 = &Test2{TestEmail: TestEmail{Email: "     test@ntnxsherlock.com"}}
		err = base.ValidateStruct("test", test2, "")
		if err == nil {
			if test2.Email == "test@ntnxsherlock.com" {
				t.Logf("Passed for %+v", test2)
			} else {
				t.Fatalf("Failed to trim the email")
			}
		} else {
			t.Fatalf("Error: %s", err.Error())
		}
		// Email negative test
		test2 = &Test2{TestEmail: TestEmail{Email: "test$ntnxsherlock.com"}}
		err = base.ValidateStruct("test", test2, "")
		require.Errorf(t, err, "Failed for %+v", test2)
		t.Logf("Error: %s", err.Error())

		// Email negative test
		test2 = &Test2{TestEmail: TestEmail{Email: "t@y"}}
		err = base.ValidateStruct("test", test2, "")
		require.Errorf(t, err, "Failed for %+v", test2)
		t.Logf("Error: %s", err.Error())

		// Integer range test
		test3 := &Test3{TestRangeInt: TestRangeInt{I: 6}}
		err = base.ValidateStruct("test", test3, "")
		require.NoError(t, err)
		t.Logf("Passed for %+v", test3)

		// Integer range negative test
		test3 = &Test3{TestRangeInt: TestRangeInt{I: 101}}
		err = base.ValidateStruct("test", test3, "")
		require.Errorf(t, err, "Failed for %+v", test3)
		t.Logf("Error: %s", err.Error())

		// Slice range test
		str := "idValue"
		test4 := &Test4{Slices: []TestSliceRange{{ID: &str}}}
		err = base.ValidateStruct("test", test4, "")
		if err == nil {
			t.Logf("Passed for %+v", test4)
		} else {
			t.Fatalf("Error: %s", err.Error())
		}
		// Slice range negative test
		test4 = &Test4{Slices: []TestSliceRange{}}
		err = base.ValidateStruct("test", test4, "")
		require.Errorf(t, err, "Failed for %+v", test4)
		t.Logf("Error: %s", err.Error())

		// Slice range negative test on slice element
		str = "i"
		test4 = &Test4{Slices: []TestSliceRange{{ID: &str}}}
		err = base.ValidateStruct("test", test4, "")
		require.Errorf(t, err, "Failed for %+v", test4)
		t.Logf("Error: %s", err.Error())

		// String options test
		test5 := &Test5{Type: "AWS"}
		err = base.ValidateStruct("test", test5, "")
		if err == nil {
			t.Logf("Passed for %+v", test5)
		} else {
			t.Fatalf("Error: %s", err.Error())
		}
		// String options negative test
		test5 = &Test5{Type: "ABC"}
		err = base.ValidateStruct("test", test5, "")
		require.Errorf(t, err, "Failed for %+v", test5)
		t.Logf("Error: %s", err.Error())

		// Ignore test
		// String options test
		test5 = &Test5{Type: "ABC"}
		err = base.ValidateStruct("test", test5, "op1")
		require.NoError(t, err)
		t.Logf("Passed for %+v", test5)
	})

	t.Run("RedactJSON test", func(t *testing.T) {
		t.Log("running JSON redactor test")
		jsonStr1 := `{
			"xi_role": [
				{
					"account_approved": true,
					"roles": [
						{
							"password": "internal-tenant-admin"
						},
						{
							"name": "xi-iot-admin"
						},
						{
							"name": "account-admin",
							"test": ["123", "456", {"password": "value", "test": 789}]
						}
					],
					"tenant-domain": "ca1d7dda-7a82-408e-bc0c-c842eee190ec",
					"tenant-name": "",
					"password": "test",
					"tenant-properties": {
						"tenant-uuid": "ca1d7dda-7a82-408e-bc0c-c842eee190ec"
					},
					"tenant-status": "PROVISIONED"
				}
			],
			"password": "test",
			"awsCredential": {
				"accessKey": "aa",
				"secret": "aaaaaa"
			},
			"name": "blahbla1ah",
			"type": "AWS",
			"password": [1, 2],
			"test": "test"
		}`
		jsonStr2 := `{
			"password": {
			  "accessKey": "aa",
			  "secret": "striaang"
			},
			"name": "asadadad",
			"type": "AWS"
		}`
		jsonStr3 := `{
			"password": {
				"accessKey": "aa",
				"secret": "striaang"
			},
			"name": "line1\nline2\nline3",
			"type": "AWS"
		}`
		redactPredicate := func(property string) bool {
			return property == "password"
		}
		redactedJSON := base.RedactJSON(jsonStr1, 0, redactPredicate)
		t.Logf("Redacted JSON: %s", redactedJSON)
		verifyRedactedJSON(t, redactedJSON, redactPredicate)
		redactedJSON = base.RedactJSON(jsonStr2, 0, redactPredicate)
		t.Logf("Redacted JSON: %s", redactedJSON)
		verifyRedactedJSON(t, redactedJSON, redactPredicate)
		invalidInput := "hello, {"
		redactedJSON = base.RedactJSON(invalidInput, 0, redactPredicate)
		t.Logf("Redacted JSON: %s", redactedJSON)
		if redactedJSON != invalidInput {
			t.Fatalf("Expected %s but got %s", invalidInput, redactedJSON)
		}
		redactedJSON = base.RedactJSON(jsonStr3, 0, redactPredicate)
		var obj map[string]interface{}
		err := json.Unmarshal([]byte(redactedJSON), &obj)
		require.NoError(t, err)
		expected := "line1[n]line2[n]line3"
		t.Logf("Redacted JSON: %s", redactedJSON)
		if obj["name"] != expected {
			t.Fatalf("Expected for name %s but got %s", expected, obj["name"])
		}
	})

	t.Run("Password generator test", func(t *testing.T) {
		t.Log("running password generator test")
		for i := 0; i < 30; i++ {
			password := base.GenerateStrongPassword()
			err := model.ValidatePassword(password)
			require.NoError(t, err)
		}
	})
}

func verifyRedactedJSON(t *testing.T, jsonStr string, redactPredicate func(property string) bool) {
	var obj map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &obj)
	require.NoError(t, err, "Error in unmarshalling")
	verifyMap(t, obj, redactPredicate)
}

// RedactSlice redacts values for object keys in the slice that satisfy the predicate
func verifySlice(t *testing.T, obj []interface{}, redactPredicate func(property string) bool) {
	for _, elm := range obj {
		kind := reflect.TypeOf(elm).Kind()
		if kind == reflect.Map {
			verifyMap(t, elm.(map[string]interface{}), redactPredicate)
		} else if kind == reflect.Slice {
			slice := elm.([]interface{})
			verifySlice(t, slice, redactPredicate)
		}
	}
}

// RedactMap redacts values for object keys in the map that satisfy the predicate
func verifyMap(t *testing.T, obj map[string]interface{}, redactPredicate func(property string) bool) {
	for key, val := range obj {
		if redactPredicate(key) {
			if val != "REDACTED" {
				t.Fatalf("Value %s for key %s is not redacted", val, key)
			}
		} else {
			kind := reflect.TypeOf(val).Kind()
			if kind == reflect.Map {
				verifyMap(t, val.(map[string]interface{}), redactPredicate)
			} else if kind == reflect.Slice {
				slice := val.([]interface{})
				verifySlice(t, slice, redactPredicate)
			}
		}
	}
}

func TestIPv4Validation(t *testing.T) {
	t.Run("IPv4 validation test", func(t *testing.T) {
		t.Log("running IPv4 validation test")

		validIPs := []string{
			"192.168.0.0",
			"1.1.1.1",
			"0.0.0.0",
			"255.255.255.255",
		}
		for _, ip := range validIPs {
			if !base.IsValidIP4(ip) {
				t.Fatalf("expect IP %s to be valid", ip)
			}
		}

		inValidIPs := []string{
			"192.168.a.0",
			"192.168.1.256",
			"192.168.1.",
			"192.168.1.25.",
			"192.168..1.256",
		}
		for _, ip := range inValidIPs {
			if base.IsValidIP4(ip) {
				t.Fatalf("expect IP %s to be invalid", ip)
			}
		}

	})
}

func TestTruncateStringMaybe(t *testing.T) {
	t.Run("TruncateStringMaybe test", func(t *testing.T) {
		t.Log("running TruncateStringMaybe test")

		max := 20
		vals := []*string{
			nil,
			base.StringPtr(""),
			base.StringPtr("hello darkness my"),
			base.StringPtr("hello darkness my old"),
			base.StringPtr("hello darkness my old friend"),
		}
		expected := []*string{
			nil,
			base.StringPtr(""),
			base.StringPtr("hello darkness my"),
			base.StringPtr("[21]hello darknes..."),
			base.StringPtr("[28]hello darknes..."),
		}
		max2 := 24
		expected2 := []*string{
			nil,
			base.StringPtr(""),
			base.StringPtr("hello darkness my"),
			base.StringPtr("hello darkness my old"),
			base.StringPtr("[28]hello darkness my..."),
		}
		for i, s := range vals {
			st := base.TruncateStringMaybe(s, max)
			exp := expected[i]
			if st != exp && (st == nil || exp == nil || *st != *exp) {
				if s == nil {
					t.Fatal("truncate failed for '<nil>'\n")
				} else {
					t.Fatalf("truncate failed for '%s'", *s)
				}
			}
		}
		// no truncation for max < 20
		for i, s := range vals {
			st := base.TruncateStringMaybe(s, 19)
			exp := vals[i]
			if st != exp && (st == nil || exp == nil || *st != *exp) {
				if s == nil {
					t.Fatal("truncate failed for '<nil>'\n")
				} else {
					t.Fatalf("truncate failed for '%s'", *s)
				}
			}
		}
		for i, s := range vals {
			st := base.TruncateStringMaybe(s, max2)
			exp := expected2[i]
			if st != exp && (st == nil || exp == nil || *st != *exp) {
				if s == nil {
					t.Fatal("truncate failed for '<nil>'\n")
				} else {
					t.Fatalf("truncate failed for '%s'", *s)
				}
			}
		}
		// utf8 char
		vals3 := []*string{
			nil,
			base.StringPtr(""),
			base.StringPtr("會當凌絕頂"),
			base.StringPtr("會當凌絕頂，一覽"),
			base.StringPtr("會當凌絕頂，一覽眾山小"),
		}
		expected3 := []*string{
			nil,
			base.StringPtr(""),
			base.StringPtr("會當凌絕頂"),
			base.StringPtr("[24]會當凌絕..."),
			base.StringPtr("[33]會當凌絕..."),
		}
		max3 := 20
		for i, s := range vals3 {
			st := base.TruncateStringMaybe(s, max3)
			exp := expected3[i]
			if st != exp && (st == nil || exp == nil || *st != *exp) {
				if s == nil {
					t.Fatal("truncate failed for '<nil>'\n")
				} else {
					t.Fatalf("truncate failed for '%s', '%s' != '%s'", *s, *st, *exp)
				}
			}
		}
	})
}

func TestMD5Hash(t *testing.T) {
	hexOut := *base.GetMD5Hash("1552447155 cxs7cqjshkqbn9bol0lhbkn6erutnd 12345")
	if hexOut != "b730bcb9449aba95550143c9b01bce19" {
		t.Fatalf("Mismatched base64 encoded MD5 hash. Expected b730bcb9449aba95550143c9b01bce19, found %s", hexOut)
	}
	b64Out := base.GetBase64URLEncodedMD5Hash("1552447155 cxs7cqjshkqbn9bol0lhbkn6erutnd 12345")
	if b64Out != "tzC8uUSaupVVAUPJsBvOGQ" {
		t.Fatalf("Mismatched base64 encoded MD5 hash. Expected tzC8uUSaupVVAUPJsBvOGQ, found %s", b64Out)
	}
}

func TestGenerateShortID(t *testing.T) {
	testCases := []struct {
		len     int
		letters string
	}{
		{len: 0, letters: "a"},
		{len: 1, letters: "ab"},
		{len: 8, letters: "abcdefghij123457890"},
	}

	for _, testCase := range testCases {
		res := base.GenerateShortID(testCase.len, testCase.letters)
		if len(res) != testCase.len {
			t.Fatalf("Expected the length to be %d, but got %d", testCase.len, len(res))
		}

		// Check that all runes are present in letters
		for _, r := range res {
			if !strings.ContainsRune(testCase.letters, r) {
				t.Fatalf("Expected run %c to be present", r)
			}
		}
	}
}

func TestGenerateShortIDUniqueness(t *testing.T) {
	if os.Getenv("SECOND_CALL") != "true" {
		// spin up a child process running the exact same test
		cmd := exec.Command("go", "test", "-v", "cloudservices/common/base", "-run", "TestGenerateShortIDUniqueness")
		cmd.Env = append(os.Environ(), "SECOND_CALL=true")
		cmd.Run()
	}

	shortIDs := make([]string, 10)
	for i := 0; i < 10; i++ {
		shortIDs[i] = base.GenerateShortID(6, "abcdefghij123457890")
	}

	// If this is a child process, then write all short IDs to a file
	if os.Getenv("SECOND_CALL") == "true" {
		os.MkdirAll("testdata", 0755)
		f, _ := os.Create(path.Join("testdata", "second.txt"))
		for _, s := range shortIDs {
			f.WriteString(s)
			f.WriteString("\n")
		}
		f.Close()
	} else {
		// Else, read the output of the child process
		content, err := ioutil.ReadFile("testdata/second.txt")
		require.NoError(t, err)
		secondShortIDs := strings.Split(string(content), "\n")
		m := make(map[string]bool)
		for _, s := range secondShortIDs {
			m[s] = true
		}

		// Assert that no ID is common in  the parent and child process
		for _, s := range shortIDs {
			if m[s] {
				t.Fatalf("Found a conflict for short ID: %s", s)
			}
		}
	}
}

func TestSubstituteValues(t *testing.T) {
	input := map[string]interface{}{
		"url":          "https://user1.xiiot.com/video/hls/{{.token}}/{{.expiry}}/live.m3u8",
		"secret":       "test-secret",
		"token":        "test-token",
		"expiry":       12345,
		"clientSecret": "{{.abc}}",
		"abc":          "abcd123",
	}
	expectedOut := map[string]interface{}{
		"url":          "https://user1.xiiot.com/video/hls/test-token/12345/live.m3u8",
		"secret":       "test-secret",
		"token":        "test-token",
		"expiry":       12345,
		"clientSecret": "abcd123",
		"abc":          "abcd123",
	}
	// Emit all map entries
	output, err := base.SubstituteValues(context.Background(), input, nil)
	require.NoError(t, err)
	if !reflect.DeepEqual(expectedOut, output) {
		t.Fatalf("Mismatched maps. Expected %+v, found %+v", expectedOut, output)
	}
	expectedOut = map[string]interface{}{
		"url":          "https://user1.xiiot.com/video/hls/test-token/12345/live.m3u8",
		"token":        "test-token",
		"expiry":       12345,
		"clientSecret": "abcd123",
		"abc":          "abcd123",
	}
	// Emit all map entries except secret which is a constant
	output, err = base.SubstituteValues(context.Background(), input, func(key string) bool {
		return "secret" != key
	})
	require.NoError(t, err)
	if !reflect.DeepEqual(expectedOut, output) {
		t.Fatalf("Mismatched maps. Expected %+v, found %+v", expectedOut, output)
	}
	expectedOut = map[string]interface{}{
		"secret":       "test-secret",
		"token":        "test-token",
		"expiry":       12345,
		"clientSecret": "abcd123",
		"abc":          "abcd123",
	}
	// Emit all map entries except url which is a variable
	output, err = base.SubstituteValues(context.Background(), input, func(key string) bool {
		return "url" != key
	})
	require.NoError(t, err)
	if !reflect.DeepEqual(expectedOut, output) {
		t.Fatalf("Mismatched maps. Expected %+v, found %+v", expectedOut, output)
	}

	input = map[string]interface{}{
		"url":          "https://user1.xiiot.com/video/hls/{{.token}}/{{.expiry}}/live.m3u8",
		"secret":       "test-secret",
		"token":        "test-token",
		"expiry":       12345,
		"clientSecret": "{{.abc}}",
		"abc":          "abcd123",
		"mytopic": map[string]interface{}{
			"url": "https://user2.xiiot.com/video/hls/{{.token}}/{{.expiry}}/live.m3u8",
		},
	}
	expectedOut = map[string]interface{}{
		"secret":       "test-secret",
		"token":        "test-token",
		"expiry":       12345,
		"clientSecret": "abcd123",
		"abc":          "abcd123",
		"mytopic": map[string]interface{}{
			"url": "https://user2.xiiot.com/video/hls/test-token/12345/live.m3u8",
		},
	}
	// Test for nested field referring to outside value
	output, err = base.SubstituteValues(context.Background(), input, func(key string) bool {
		return "url" != key
	})
	require.NoError(t, err)
	if !reflect.DeepEqual(expectedOut, output) {
		t.Fatalf("Mismatched maps. Expected %+v, found %+v", expectedOut, output)
	}

	input = map[string]interface{}{
		"url":          "https://user1.xiiot.com/video/hls/{{.token}}/{{.expiry}}/live.m3u8",
		"secret":       "test-secret",
		"token":        "test-token",
		"expiry":       12345,
		"clientSecret": "{{.abc__def}}",
		"abc": map[string]interface{}{
			"def": "abcd123",
		},
		"mytopic": map[string]interface{}{
			"url": "https://user2.xiiot.com/video/hls/{{.token}}/{{.expiry}}/live.m3u8",
		},
	}
	expectedOut = map[string]interface{}{
		"secret":       "test-secret",
		"token":        "test-token",
		"expiry":       12345,
		"clientSecret": "abcd123",
		"abc": map[string]interface{}{
			"def": "abcd123",
		},
		"mytopic": map[string]interface{}{
			"url": "https://user2.xiiot.com/video/hls/test-token/12345/live.m3u8",
		},
	}
	// Test for a field referring to a nested value
	output, err = base.SubstituteValues(context.Background(), input, func(key string) bool {
		return "url" != key
	})
	require.NoError(t, err)
	if !reflect.DeepEqual(expectedOut, output) {
		t.Fatalf("Mismatched maps. Expected %+v, found %+v", expectedOut, output)
	}

	input = map[string]interface{}{
		"url":          "https://user1.xiiot.com/video/hls/{{.token}}/{{.expiry}}/live.m3u8",
		"secret":       "test-secret",
		"token":        "test-token",
		"expiry":       12345,
		"clientSecret": "{{.abc__def}}",
		"abc": map[string]interface{}{
			"def": "abcd123",
		},
		"mytopic": map[string]interface{}{
			"url": "https://user2.xiiot.com/video/hls/{{.token}}/{{.expiry}}/live.m3u8",
		},
	}
	expectedOut = map[string]interface{}{
		"secret":       "test-secret",
		"token":        "test-token",
		"expiry":       12345,
		"clientSecret": "abcd123",
		"mytopic": map[string]interface{}{
			"url": "https://user2.xiiot.com/video/hls/test-token/12345/live.m3u8",
		},
	}
	// Test for excluding a nested field
	output, err = base.SubstituteValues(context.Background(), input, func(key string) bool {
		if "url" == key {
			return false
		}
		return key != "abc__def"
	})
	require.NoError(t, err)
	if !reflect.DeepEqual(expectedOut, output) {
		t.Fatalf("Mismatched maps. Expected %+v, found %+v", expectedOut, output)
	}

	input = map[string]interface{}{
		"token":  "test-token",
		"expiry": 12345,
		"endpoints": []interface{}{
			map[string]interface{}{
				"name": "channel1",
				"url":  "https://34.221.86.34:30071/video/hls/{{.token}}/{{.expiry}}/channel1.m3u8",
			},
			map[string]interface{}{
				"name": "channel2",
				"url":  "https://34.221.86.34:30071/video/hls/{{.token}}/{{.expiry}}/channel2.m3u8",
			},
			map[string]interface{}{
				"name": "channel2",
				"url":  "https://34.221.86.34:{{.ports__0}}/video/hls/test-token/12345/channel2.m3u8",
			},
		},
		"ports": []interface{}{
			"3000",
		},
	}
	expectedOut = map[string]interface{}{
		"token":  "test-token",
		"expiry": 12345,
		"endpoints": []interface{}{
			map[string]interface{}{
				"name": "channel1",
				"url":  "https://34.221.86.34:30071/video/hls/test-token/12345/channel1.m3u8",
			},
			map[string]interface{}{
				"name": "channel2",
				"url":  "https://34.221.86.34:30071/video/hls/test-token/12345/channel2.m3u8",
			},
			map[string]interface{}{
				"name": "channel2",
				"url":  "https://34.221.86.34:3000/video/hls/test-token/12345/channel2.m3u8",
			},
		},
	}
	// Test for excluding a nested field
	output, err = base.SubstituteValues(context.Background(), input, func(key string) bool {
		return "ports__0" != key
	})
	require.NoError(t, err)
	if !reflect.DeepEqual(expectedOut, output) {
		t.Fatalf("Mismatched maps. Expected %+v, found %+v", expectedOut, output)
	}

}

func TestEncodeDecodeTokens(t *testing.T) {
	tokenTests := []struct {
		in  []string
		out string
	}{
		{[]string{"first-token-untouched", "second.token", "3rd", "", "5th"}, "first-token-untouched.c2Vjb25kLnRva2Vu.M3Jk..NXRo"},
		{[]string{"first-token-untouched", "second.token"}, "first-token-untouched.c2Vjb25kLnRva2Vu"},
	}
	for _, tt := range tokenTests {
		t.Run(tt.out, func(t *testing.T) {
			s := base.EncodeTokens(tt.in, 1)
			if s != tt.out {
				t.Errorf("got %q, want %q", s, tt.out)
			}
			dts, err := base.DecodeTokens(s, 1)
			require.NoError(t, err)
			if !reflect.DeepEqual(dts, tt.in) {
				t.Errorf("got %q, want %q", dts, tt.in)
			}
		})
	}
}

func TestTraverseDependencies(t *testing.T) {
	dependencies := map[string]map[string]bool{
		"category_model": {
			"category_value_model": true,
		},
		"data_source_model": {
			"data_source_field_model": true,
			"category_model":          true,
		},
		"tenant_model": {
			"user_model":           true,
			"category_model":       true,
			"data_source_model":    true,
			"cloud_creds_model":    true,
			"docker_profile_model": true,
		},
	}
	count := 0
	err := base.TraverseDependencies(dependencies, func(deletableTables map[string]bool) error {
		if count == 0 {
			expectedTables := []string{
				"category_value_model",
				"data_source_field_model",
				"user_model",
				"cloud_creds_model",
				"docker_profile_model",
			}
			tableVerifier(t, expectedTables, deletableTables)
		} else if count == 1 {
			expectedTables := []string{
				"category_model",
			}
			tableVerifier(t, expectedTables, deletableTables)
		} else if count == 2 {
			expectedTables := []string{
				"data_source_model",
			}
			tableVerifier(t, expectedTables, deletableTables)
		} else if count == 3 {
			expectedTables := []string{
				"tenant_model",
			}
			tableVerifier(t, expectedTables, deletableTables)
		} else if count == 4 {
			t.Fatal("No more call expected")
		}
		count++
		return nil
	})
	require.NoError(t, err)
}

func tableVerifier(t *testing.T, expectedTables []string, deletableTables map[string]bool) {
	_, file, line, _ := runtime.Caller(1)
	caller := fmt.Sprintf("%s:%d", file, line)
	if len(deletableTables) != len(expectedTables) {
		t.Fatalf("Expected %d, found %d. From %s", len(expectedTables), len(deletableTables), caller)
	}
	for _, table := range expectedTables {
		if _, ok := deletableTables[table]; !ok {
			t.Fatalf("Expected table %s not found in %+v. From %s", table, deletableTables, caller)
		}
	}
}

func TestCheckID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{"Empty id", "", false},
		{"Short id", strings.Repeat("a", 7), false},
		{"Short ok id", strings.Repeat("a", 8), true},
		{"Long ok id", strings.Repeat("a", 36), true},
		{"Long id", strings.Repeat("a", 37), false},
		{"Forbidden chars", "abcdefg", false},
		{"UUID", uuid.Generate().String(), true},
		{"Custom correct ID", "abcdefg-123", true},
		{"Email", "te.st@nutanix-test.com", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := base.CheckID(tt.id); got != tt.want {
				t.Errorf("CheckID(%s) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestUUID(t *testing.T) {
	tests := []struct {
		ba   []byte
		uuid string
	}{
		{nil, "00000000-0000-0000-0000-000000000000"},
		{[]byte{0}, "00000000-0000-0000-0000-000000000000"},
		{[]byte{0x01}, "00000000-0000-0000-0000-000000000001"},
		{[]byte{0x01, 0x02}, "00000000-0000-0000-0000-000000000102"},
		{[]byte{0x01, 0xFF}, "00000000-0000-0000-0000-0000000001ff"},
	}
	for _, tt := range tests {
		uuid, err := base.GetUUIDFromBytes(tt.ba)
		require.NoError(t, err)
		if uuid != tt.uuid {
			t.Fatalf("expected %s, found %s", tt.uuid, uuid)
		}
	}
}
