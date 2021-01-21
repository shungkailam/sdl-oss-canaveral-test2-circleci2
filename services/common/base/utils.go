package base

import (
	"bytes"
	"cloudservices/common/errcode"
	"context"
	"crypto/md5"
	crypto_rand "crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"math/rand"
	"mime/multipart"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/golang/glog"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
)

type JSONTokenType int

const (
	ObjectToken JSONTokenType = iota
	ArrayToken

	ValidateTag        = "validate"
	ValidateIgnoreTag  = "ignore"
	ValidateRangeTag   = "range"
	ValidateNonZeroTag = "non-zero"
	ValidateEmailTag   = "email"
	ValidateOptionsTag = "options"
)

var (
	reIP4              = regexp.MustCompile(`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$`)
	reTemplateVariable = regexp.MustCompile(`{{.*\.[a-zA-Z0-9_]+}}`)
)

// StringArray is a type implementing the sql/driver/value interface
// This is due to the native driver not supporting arrays...
type StringArray []string

// PrefixRequestID adds the request ID prefix and tenantID (if present)
// This function makes the format uniform.
func PrefixRequestID(ctx context.Context, format string) string {
	reqID := GetRequestID(ctx)
	authContext, err := GetAuthContext(ctx)
	if err == nil {
		return fmt.Sprintf("Request %s: Tenant %s Message %s", reqID, authContext.TenantID, format)
	}
	return fmt.Sprintf("Request %s: Message %s", reqID, format)
}

func StringPtr(in string) *string {
	return &in
}

func IntPtr(in int) *int {
	return &in
}

func BoolPtr(in bool) *bool {
	return &in
}

func DurationPtr(in time.Duration) *time.Duration {
	return &in
}

func TimePtr(in time.Time) *time.Time {
	return &in
}

func Float64Ptr(in float64) *float64 {
	return &in
}

func RoundedNow() time.Time {
	return time.Now().UTC().Round(time.Microsecond)
}

// GetUUID creates new UUID
func GetUUID() string {
	return uuid.New().String()
}

// GetUUIDFromBytes gets the UUID string with the LSBs filled from the input byte slice
func GetUUIDFromBytes(ba []byte) (string, error) {
	var uuidStr string
	// UUID is 16 bytes, each byte is represented by two characters
	ba16 := make([]byte, 16, 16)
	baLen := len(ba)
	if baLen >= 16 {
		copy(ba16[0:16], ba)
	} else {
		copy(ba16[16-baLen:16], ba)
	}
	gid, err := uuid.FromBytes(ba16)
	if err == nil {
		uuidStr = gid.String()
	}
	return uuidStr, err
}

// MustGetUUIDFromBytes gets the UUID string with the LSBs filled from the input byte slice.
// It panics on error
func MustGetUUIDFromBytes(ba []byte) string {
	uuid, err := GetUUIDFromBytes(ba)
	if err != nil {
		panic(err)
	}
	return uuid
}

// CheckID checks if the string is UUID
func CheckID(id string) bool {
	match, err := regexp.Match(`^[A-Za-z0-9\-@\.]{8,36}$`, []byte(id))
	return match && err == nil
}

// GetMD5Hash returns MD5 of the string in hex format
func GetMD5Hash(data string) *string {
	hasher := md5.New()
	hasher.Write([]byte(data))
	hash := hex.EncodeToString(hasher.Sum(nil))
	return &hash
}

// GetBase64URLEncodedMD5Hash returns the base64 URL encoded MD5 hash
func GetBase64URLEncodedMD5Hash(data string) string {
	hasher := md5.New()
	hasher.Write([]byte(data))
	return strings.TrimRight(base64.URLEncoding.EncodeToString(hasher.Sum(nil)), "=")
}

// DispatchPayload calls DispatchListPayload if it is a slice otherwise the object is directly marshaled and sent.
func DispatchPayload(w io.Writer, response interface{}) error {
	value := reflect.ValueOf(response)
	pType := value.Type()
	if pType.Kind() != reflect.Slice {
		encoder := json.NewEncoder(w)
		return encoder.Encode(response)
	}
	idx := 0
	length := value.Len()
	return DispatchListPayload(w, func() (interface{}, error) {
		if length == idx {
			return nil, io.EOF
		}
		item := value.Index(idx).Interface()
		idx++
		return item, nil
	})
}

// DispatchListPayload sends streaming array objects.
func DispatchListPayload(w io.Writer, source func() (interface{}, error)) error {
	var payload interface{}
	var err error
	firstObject := true
	_, err = w.Write([]byte("["))
	encoder := json.NewEncoder(w)
	for {
		payload, err = source()
		if err == io.EOF {
			err = nil
			_, err = w.Write([]byte("]"))
			break
		}
		if err == nil {
			if firstObject {
				firstObject = false
			} else {
				_, err = w.Write([]byte(","))
			}
			if err == nil {
				err = encoder.Encode(payload)
			}
		}
		if err != nil {
			break
		}
	}
	return err
}

// Decode decodes the JSOn string from the reader into doc.
func Decode(r *io.Reader, doc interface{}) error {
	decoder := json.NewDecoder(*r)
	err := decoder.Decode(&doc)
	if err != nil {
		glog.Errorf("Error in decoding. Error: %s", err.Error())
		return errcode.NewDataConversionError(err.Error())
	}
	return nil
}

// Convert converts from one object to another compatible object.
func Convert(from interface{}, to interface{}) error {
	data, err := ConvertToJSON(from)
	if err != nil {
		return err
	}
	return ConvertFromJSON(data, to)
}

// ConvertFromJSON converts from JSON string to an object which can be a protobuf type.
func ConvertFromJSON(jsonData []byte, to interface{}) error {
	var err error
	toMsg, ok := to.(proto.Message)
	if ok {
		err = jsonpb.UnmarshalString(string(jsonData), toMsg)
	} else {
		err = json.Unmarshal(jsonData, to)
	}
	if err != nil {
		return errcode.NewDataConversionError(fmt.Sprintf("Unable to convert from JSON to object. Error: %s", err.Error()))
	}
	return nil
}

// ConvertToJSON converts an object which can be a protobuf type to a JSON string.
func ConvertToJSON(from interface{}) ([]byte, error) {
	var data []byte
	var err error
	fromMsg, ok := from.(proto.Message)
	if ok {
		marshaller := jsonpb.Marshaler{}
		jstr, err := marshaller.MarshalToString(fromMsg)
		if err != nil {
			return nil, errcode.NewDataConversionError(fmt.Sprintf("Unable to convert object to JSON. Error: %s", err.Error()))
		}
		data = []byte(jstr)
	} else {
		data, err = json.Marshal(from)
		if err != nil {
			return nil, errcode.NewDataConversionError(fmt.Sprintf("Unable to convert from JSON to object. Error: %s", err.Error()))
		}
	}
	return data, nil
}

// ConvertToJSONIndent converts an object which can be a protobuf type to an indented JSON string.
func ConvertToJSONIndent(from interface{}, indent string) ([]byte, error) {
	var data []byte
	var err error
	fromMsg, ok := from.(proto.Message)
	if ok {
		marshaller := jsonpb.Marshaler{Indent: indent}
		jstr, err := marshaller.MarshalToString(fromMsg)
		if err != nil {
			return nil, errcode.NewDataConversionError(fmt.Sprintf("Unable to convert object to JSON. Error: %s", err.Error()))
		}
		data = []byte(jstr)
	} else {
		data, err = json.MarshalIndent(from, "", indent)
		if err != nil {
			return nil, errcode.NewDataConversionError(fmt.Sprintf("Unable to convert from JSON to object. Error: %s", err.Error()))
		}
	}
	return data, nil
}

// Call invokes the callback with timneout.
func Call(ctx context.Context, callback func(context.Context) error, timeout time.Duration) error {
	reqID := GetRequestID(ctx)
	newCtx, cancelFunc := context.WithTimeout(ctx, timeout)
	defer cancelFunc()
	ch := make(chan error, 1)
	var err error
	go func(ctx context.Context) {
		err := callback(ctx)
		if err != nil {
			glog.Errorf("Request %s: Error in client callback. Error: %+v", reqID, err)
		}
		ch <- err
		close(ch)
	}(newCtx)
label:
	for {
		select {
		case <-newCtx.Done():
			<-ch
			err = newCtx.Err()
			break label
		case err = <-ch:
			break label
		}
	}
	return err
}

func IsTagPresent(tagNames []string, tagName string) ([]string, bool) {
	for _, name := range tagNames {
		values := strings.Split(name, "=")
		if values[0] == tagName {
			if len(values) == 2 {
				return strings.Split(values[1], ":"), true
			}
			return []string{}, true
		}
	}
	return nil, false
}

// ValidateStruct validates a model referenced by the name for any invalid values in the fields (nested as well)
func ValidateStruct(name string, model interface{}, operation string) error {
	value := reflect.ValueOf(model)
	// The string fields are not addressable if pointer is not received.
	// https://stackoverflow.com/questions/6395076/using-reflect-how-do-you-set-the-value-of-a-struct-field
	if value.Type().Kind() != reflect.Ptr {
		return errcode.NewBadRequestError(name)
	}
	if value.IsNil() {
		return errcode.NewBadRequestError(name)
	}
	return validateValue(name, value, operation)
}

// validateValue operates on the reflect value to preserve the reference
func validateValue(name string, value reflect.Value, operation string) error {
	err := validateStructHelper(reflect.Indirect(value), fieldValidator, operation)
	if err != nil {
		glog.Errorf("Validation failed for field %s. Error: %s", name, err.Error())
	}
	return err
}

// validateStructHelper retrieves the fields for validation
func validateStructHelper(value reflect.Value, validateFunc func(fieldName string, value reflect.Value, tagNames []string, operation string) error, operation string) error {
	vType := value.Type()
	if vType.Kind() != reflect.Struct {
		return nil
	}
	nField := vType.NumField()
outer:
	for i := 0; i < nField; i++ {
		sField := vType.Field(i)
		fType := sField.Type
		fName := sField.Name
		fValue := value.FieldByName(fName)
		tag, ok := sField.Tag.Lookup(ValidateTag)
		if !ok {
			// There can be tags in the inner struct, check recursively
			if fType.Kind() == reflect.Struct {
				err := validateValue(fName, fValue, operation)
				if err != nil {
					return err
				}
			} else if fType.Kind() == reflect.Slice {
				length := fValue.Len()
				// Iterate over the elements in the slice
				for i := 0; i < length; i++ {
					elem := fValue.Index(i)
					err := validateValue(fName, elem, operation)
					if err != nil {
						return err
					}
				}
			}
			// No need to analyze other field types as there is no tag
			continue
		}
		tagNames := strings.Split(tag, ",")
		if len(operation) > 0 {
			ignoreOps, ignore := IsTagPresent(tagNames, ValidateIgnoreTag)
			if ignore {
				for _, ignoreOp := range ignoreOps {
					if ignoreOp == operation {
						continue outer
					}
				}
			}
		}
		// Tag is found
		if fType.Kind() == reflect.Ptr {
			if fValue.IsNil() {
				// Reject nil pointer as they cannot be converted to interface value
				if _, yes := IsTagPresent(tagNames, ValidateNonZeroTag); yes {
					return errcode.NewBadRequestError(fName)
				}
				continue
			}
			// Avoid dealing with pointer type later as it makes complex unnecessarily
			fValue = reflect.Indirect(fValue)
		}
		if fType.Kind() == reflect.Func {
			if fValue.IsNil() {
				if _, yes := IsTagPresent(tagNames, ValidateNonZeroTag); yes {
					return errcode.NewBadRequestError(fName)
				}
			}
			continue
		}
		// Check inner structs
		if fType.Kind() == reflect.Struct {
			err := validateValue(fName, fValue, operation)
			if err != nil {
				return err
			}
		}
		// Validate the current field
		err := validateFunc(fName, fValue, tagNames, operation)
		if err != nil {
			return err
		}
	}
	return nil
}

// fieldValidator does the validation for value.
// More validations can be added here. value is never nil
func fieldValidator(fieldName string, value reflect.Value, tagNames []string, operation string) error {
	checkLengthFunc := func(fieldValue int64, isLength bool) error {
		values, yes := IsTagPresent(tagNames, ValidateRangeTag)
		if yes {
			if len(values) > 0 {
				min, err := strconv.ParseInt(values[0], 10, 64)
				if err != nil {
					return err
				}
				if fieldValue < min {
					if isLength {
						return errcode.NewMinLenBadRequestError(fieldName, strconv.FormatInt(min, 10))
					}
					return errcode.NewMinValBadRequestError(fieldName, strconv.FormatInt(min, 10))
				}
			}
			if len(values) > 1 {
				max, err := strconv.ParseInt(values[1], 10, 64)
				if err != nil {
					return err
				}
				if fieldValue > max {
					if isLength {
						return errcode.NewMaxLenBadRequestError(fieldName, strconv.FormatInt(max, 10))
					}
					return errcode.NewMaxValBadRequestError(fieldName, strconv.FormatInt(max, 10))
				}
			}
		}
		return nil
	}
	vType := value.Type()
	// These are the only handled types
	switch vType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return checkLengthFunc(value.Int(), false)

	case reflect.String:
		strVal := value.String()
		trimmedStrVal := strings.TrimSpace(strVal)
		if strVal != trimmedStrVal {
			value.SetString(trimmedStrVal)
			strVal = trimmedStrVal
		}
		err := checkLengthFunc(int64(len(strVal)), true)
		if err != nil {
			return err
		}
		_, yes := IsTagPresent(tagNames, ValidateEmailTag)
		if yes {
			if ValidateEmail(strVal) != nil {
				return errcode.NewMalformedBadRequestError(fieldName)
			}
		}
		options, yes := IsTagPresent(tagNames, ValidateOptionsTag)
		if yes {
			for _, option := range options {
				if option == strVal {
					return nil
				}
			}
			return errcode.NewWrongOptionBadRequestError(fieldName, strings.Join(options, ", "))
		}

	case reflect.Slice:
		length := value.Len()
		err := checkLengthFunc(int64(length), true)
		if err != nil {
			return err
		}
		for i := 0; i < length; i++ {
			elem := value.Index(i)
			err := validateValue(fieldName, elem, operation)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func Unique(inList []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range inList {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func And(aList, bList []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range aList {
		keys[entry] = true
	}
	for _, entry := range bList {
		if val, ok := keys[entry]; ok && val {
			keys[entry] = false
			list = append(list, entry)
		}
	}
	return list
}

// ValidateEmail is a simple email validator
func ValidateEmail(email string) error {
	tokens := strings.SplitN(strings.TrimSpace(email), "@", 3)
	if len(tokens) != 2 {
		return errcode.NewMalformedBadRequestError("email")
	}
	return nil
}

// MaskString mask out the middle of the given string
// E.g., MaskString("hello", "*", 1, 2) -> "h**lo"
// @param start How many char from start of string to not mask
// @param end How many char from end of string to not mask
func MaskString(s string, mask string, start int, end int) string {
	strLen := len(s)
	maskLen := strLen - start - end
	if start < 0 || end < 0 || maskLen <= 0 {
		return s
	}
	return fmt.Sprintf("%s%s%s", s[:start], strings.Repeat(mask, maskLen), s[strLen-end:])
}

// GetDBURL constructs the DB URL
func GetDBURL(dialect, db, user, password, host string, port int, disableSSL bool) (string, error) {
	switch dialect {
	case "postgres":
		sfx := ""
		if disableSSL {
			sfx = "?sslmode=disable"
		}
		return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s%s", user, password, host, port, db, sfx), nil
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, host, port, db), nil
	default:
		return "", errcode.NewBadRequestError(dialect)
	}
}

// RedactJSON redacts values for the fields in the JSON that satisfiy the predicate.
// It stops when the number of processed properties hits numProperties.
// If any error happens, it just returns the input JSON string.
// This works iteratively rather than unmarshalling the full JSON.
func RedactJSON(jsonStr string, numProperties int, redactPredicate func(string) bool) string {
	decoder := json.NewDecoder(strings.NewReader(jsonStr))
	var nestingStates []JSONTokenType
	var buffer bytes.Buffer
	// Currently selected property
	var currentProperty string
	// Next token is property or value
	var inProperty bool
	var propertiesCount int
	for {
		if numProperties > 0 && propertiesCount >= numProperties {
			buffer.WriteString("...truncated")
			break
		}
		token, err := decoder.Token()
		if err == io.EOF {
			break
		} else if err != nil {
			return jsonStr
		}
		if token == nil {
			continue
		}
		if delim, ok := token.(json.Delim); ok {
			delimStr := delim.String()
			switch delimStr {
			case "{":
				if redactPredicate(currentProperty) {
					eatupTokens(decoder)
					// Value is already set
					buffer.WriteString("\"REDACTED\"")
					if decoder.More() {
						// More properties (in case of object) or values (in case of slice) to come
						buffer.WriteString(",")
					}
				} else {
					buffer.WriteString(delimStr)
					nestingStates = append(nestingStates, ObjectToken)
				}
				inProperty = true
			case "[":
				if redactPredicate(currentProperty) {
					eatupTokens(decoder)
					buffer.WriteString("\"REDACTED\"")
					// Value is already set
					inProperty = true
					if decoder.More() {
						// More properties (in case of object) or values (in case of slice) to come
						buffer.WriteString(",")
					}
				} else {
					buffer.WriteString(delimStr)
					nestingStates = append(nestingStates, ArrayToken)
				}
			default:
				buffer.WriteString(delimStr)
				topIndex := len(nestingStates) - 1
				if topIndex < 0 {
					// Malformed JSON can be encountered
					return jsonStr
				}
				// Pop the top because the object is done
				nestingStates = nestingStates[0:topIndex]
				if decoder.More() {
					// More properties (in case of object) or values (in case of slice) to come
					buffer.WriteString(",")
				}
				topIndex--
				if topIndex >= 0 {
					// Find the current scope
					top := nestingStates[topIndex]
					inProperty = top == ObjectToken
				}
			}

		} else if strToken, ok := token.(string); ok {
			topIndex := len(nestingStates) - 1
			if topIndex < 0 {
				// Malformed JSON can be encountered
				return jsonStr
			}
			top := nestingStates[topIndex]
			if top == ObjectToken {
				if inProperty {
					propertiesCount++
					currentProperty = strToken
					buffer.WriteString("\"")
					buffer.WriteString(strToken)
					buffer.WriteString("\": ")
				} else {
					if redactPredicate(currentProperty) {
						strToken = "REDACTED"
					} else {
						// Replace \n with [n] to compact
						strToken = strings.Replace(strToken, "\n", "[n]", -1)
					}
					buffer.WriteString("\"")
					buffer.WriteString(strToken)
					buffer.WriteString("\"")
					if decoder.More() {
						buffer.WriteString(",")
					}
					// Ready for next property
					currentProperty = ""
				}
				// Flip
				inProperty = !inProperty
			} else if top == ArrayToken {
				// Array can be values only. No flipping
				buffer.WriteString("\"")
				// Replace \n with [n] to compact. Array values most likely may not have \n though
				buffer.WriteString(strings.Replace(strToken, "\n", "[n]", -1))
				buffer.WriteString("\"")
				if decoder.More() {
					buffer.WriteString(",")
				}
			}

		} else {
			if boolToken, ok := token.(bool); ok {
				buffer.WriteString(fmt.Sprintf("%t", boolToken))
			} else if float64Token, ok := token.(float64); ok {
				buffer.WriteString(fmt.Sprintf("%f", float64Token))
			} else {
				// Number
				buffer.WriteString(fmt.Sprintf("%d", token))
			}
			if decoder.More() {
				buffer.WriteString(",")
			}
			inProperty = true
		}
	}
	return buffer.String()
}

func eatupTokens(decoder *json.Decoder) {
	level := 1
	forward := true
	var token json.Token
	for {
		if forward && decoder.More() {
			if _, ok := token.(json.Delim); ok {
				level++
			}
		} else if level == 0 {
			break
		} else {
			forward = false
			level--
		}
		token, _ = decoder.Token()
	}
}

// GenerateStrongPassword generates password
func GenerateStrongPassword() string {
	return GenerateStrongPasswordWithLength(40)
}

// GenerateStrongPasswordWithLength generates password with given length
func GenerateStrongPasswordWithLength(length int) string {
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	la := int('a')
	lz := int('z')
	ua := int('A')
	uz := int('Z')
	d0 := int('0')
	d9 := int('9')
	special := "!@#$%^&*~"
	specialLen := len(special)
	result := []string{}
	for len(result) < length {
		// Upper bound not included, so +1
		str1 := string(random.Intn(lz-la) + la + 1)
		str2 := string(random.Intn(uz-ua) + ua + 1)
		str3 := string(random.Intn(d9-d0) + d0 + 1)
		str4 := string(special[random.Intn(specialLen)])
		result = append(result, str1, str2, str3, str4)
	}
	Shuffle(random, len(result), func(i, j int) {
		result[i], result[j] = result[j], result[i]
	})
	return strings.Join(result, "")
}

// Shuffle shuffles using the random generator
// Copied from go1.11.1 as it is not available in go1.9.3
// TODO remove when we upgrade https://github.com/golang/go/issues/20480
func Shuffle(random *rand.Rand, n int, swap func(i, j int)) {
	if random == nil || n < 0 {
		panic("invalid argument to Shuffle")
	}

	// Fisher-Yates shuffle: https://en.wikipedia.org/wiki/Fisher%E2%80%93Yates_shuffle
	// Shuffle really ought not be called with n that doesn't fit in 32 bits.
	// Not only will it take a very long time, but with 2³¹! possible permutations,
	// there's no way that any PRNG can have a big enough internal state to
	// generate even a minuscule percentage of the possible permutations.
	// Nevertheless, the right API signature accepts an int n, so handle it as best we can.
	i := n - 1
	for ; i > 1<<31-1-1; i-- {
		j := int(random.Int63n(int64(i + 1)))
		swap(i, j)
	}
	for ; i > 0; i-- {
		j := int(random.Int31n(int32(i + 1)))
		swap(i, j)
	}
}

// TruncateStringMaybe truncate s if its length exceeds max, to the form:
// [<length>]<prefix of s up to max - len[...] - 3>...
// E.g. s := "hello world foo bar baz great to see you"
// then the value of TruncateStringMaybe(&s, 24) would be
// "[40]hello world foo b..."
func TruncateStringMaybe(s *string, max int) *string {
	if s == nil {
		return nil
	}
	n := len(*s)
	if n <= max || max < 20 {
		return s
	}
	prefix := fmt.Sprintf("[%d]", len(*s))
	substr := (*s)[:(max - len(prefix) - 3)]
	for !utf8.ValidString(substr) {
		substr = substr[:len(substr)-1]
	}
	sTruncated := fmt.Sprintf("%s%s...", prefix, substr)
	return &sTruncated
}

// GenerateShortID generates short ID of length `n` from the given letters
func GenerateShortID(n int, letters string) string {
	output := make([]byte, n)

	// Take n bytes, one byte for each character of output.
	randomness := make([]byte, n)
	// read all random
	// Intenationally, not handling errors from rand.Read() as it always returns nil as per documentation.
	crypto_rand.Read(randomness)

	l := len(letters)

	// fill output
	for i := range output {
		// get random item
		random := uint8(randomness[i])
		// Get the position of next char in
		randomPos := random % uint8(l)
		// put into output
		output[i] = letters[randomPos]
	}
	return string(output)
}

// GenerateRandomSecret returns a randomly generated secret with the input length
func GenerateRandomSecret(length int) string {
	return GenerateShortID(length, "abcdefghijklmnopqrstuvwxyz0123456789")
}

// SubstituteValues substitutes the variables (map key) in string values from the values in other keys.
// The predicate is applied to each input key to include or exclude in the output.
// A variable is inserted as {{.variableName}}. Nested keys are separated by __
func SubstituteValues(ctx context.Context, input map[string]interface{}, predicate func(key string) bool) (map[string]interface{}, error) {
	var ok bool
	var outputArtifact map[string]interface{}
	substitues := map[string]interface{}{}
	extractConstants(ctx, input, []string{}, substitues)
	output := substituteVariables(ctx, input, []string{}, substitues, predicate)
	if output != nil {
		if outputArtifact, ok = output.(map[string]interface{}); ok {
			return outputArtifact, nil
		}
	}
	return map[string]interface{}{}, nil
}

// extractConstants extracts the constants from the input of map type into the output map
// The output map contains the flattened key value pairs with nesting level indicated by __
func extractConstants(ctx context.Context, input interface{}, prefixes []string, output map[string]interface{}) {
	v := reflect.ValueOf(input)
	if v.Kind() == reflect.Map {
		for _, key := range v.MapKeys() {
			val := v.MapIndex(key)
			prefixes = append(prefixes, key.String())
			extractConstants(ctx, val.Interface(), prefixes, output)
			prefixes = prefixes[:len(prefixes)-1]
		}
	} else if v.Kind() == reflect.Slice {
		for i := 0; i < v.Len(); i++ {
			item := v.Index(i)
			prefixes = append(prefixes, fmt.Sprintf("%d", i))
			extractConstants(ctx, item.Interface(), prefixes, output)
			prefixes = prefixes[:len(prefixes)-1]
		}
	} else {
		// Slice types are not handled for values
		switch v.Kind() {
		case reflect.String:
			str := input.(string)
			if !reTemplateVariable.MatchString(str) {
				// Create key value pair without nesting for substitution
				output[strings.Join(prefixes, "__")] = input
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			output[strings.Join(prefixes, "__")] = input
		default:
			glog.Warningf(PrefixRequestID(ctx, "Unhandled kind %s"), v.Kind())
		}

	}
}

// substituteVariables substitues the template variables with the values in substitutes
func substituteVariables(ctx context.Context, input interface{}, prefixes []string, substitues map[string]interface{}, predicate func(key string) bool) interface{} {
	v := reflect.ValueOf(input)
	if v.Kind() == reflect.Map {
		outputMap := map[string]interface{}{}
		for _, key := range v.MapKeys() {
			val := v.MapIndex(key)
			prefixes = append(prefixes, key.String())
			value := substituteVariables(ctx, val.Interface(), prefixes, substitues, predicate)
			if value != nil {
				outputMap[key.Interface().(string)] = value
			}
			prefixes = prefixes[:len(prefixes)-1]
		}
		if len(outputMap) == 0 {
			return nil
		}
		return outputMap
	} else if v.Kind() == reflect.Slice {
		outputSlice := []interface{}{}
		for i := 0; i < v.Len(); i++ {
			item := v.Index(i)
			prefixes = append(prefixes, fmt.Sprintf("%d", i))
			value := substituteVariables(ctx, item.Interface(), prefixes, substitues, predicate)
			if value != nil {
				outputSlice = append(outputSlice, value)
			}
			prefixes = prefixes[:len(prefixes)-1]
		}
		if len(outputSlice) == 0 {
			return nil
		}
		return outputSlice
	} else {
		value := input
		key := strings.Join(prefixes, "__")
		if predicate == nil || predicate(key) {
			if v.Kind() == reflect.String {
				str := input.(string)
				if reTemplateVariable.MatchString(str) {
					t, err := template.New(key).Parse(str)
					if err != nil {
						glog.Errorf(PrefixRequestID(ctx, "Failed to parse template %s for key %s. Error: %s"), str, key, err.Error())
						return errcode.NewBadRequestError(key)
					}
					var w bytes.Buffer
					err = t.Execute(&w, substitues)
					if err != nil {
						glog.Errorf(PrefixRequestID(ctx, "Failed to substitute for input key %s. Error: %s"), key, err.Error())
						return errcode.NewBadRequestError(key)
					}
					value = w.String()
				}
			}
			return value
		}
	}
	return nil
}

// ReaderWrapper is a simple io.Reader wrapper to track content length with the max length of bytes
type ReaderWrapper struct {
	io.Reader
	length   int64
	maxBytes int64
}

func NewReaderWrapper(r io.Reader) *ReaderWrapper {
	return &ReaderWrapper{r, 0, 0}
}

func NewReaderWrapperWithLimit(r io.Reader, maxBytes int64) *ReaderWrapper {
	return &ReaderWrapper{r, 0, maxBytes}
}

func (r *ReaderWrapper) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	if err == nil || err == io.EOF {
		r.length += int64(n)
		if r.maxBytes > 0 && r.length > r.maxBytes {
			err = multipart.ErrMessageTooLarge
		}
	}
	return
}

func (r *ReaderWrapper) Len() int64 {
	return r.length
}

func EncodeTokens(tokens []string, startIndex int) string {
	encodedTokens := []string{}
	for i, token := range tokens {
		if i >= startIndex {
			token = base64.StdEncoding.EncodeToString([]byte(token))
		}
		encodedTokens = append(encodedTokens, token)
	}
	return strings.Join(encodedTokens, ".")
}

func DecodeTokens(s string, startIndex int) ([]string, error) {
	encodedTokens := strings.Split(s, ".")
	tokens := []string{}
	for i, encodedToken := range encodedTokens {
		token := encodedToken
		if i >= startIndex {
			ba, err := base64.StdEncoding.DecodeString(encodedToken)
			if err != nil {
				return nil, err
			}
			token = string(ba)
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}

// TraverseDependencies traversers the dependencies map (DAG) and
// invokes the callback with the entries starting from the deepest levels in the DAG
func TraverseDependencies(dependencies map[string]map[string]bool, callback func(map[string]bool) error) error {
	for {
		if len(dependencies) == 0 {
			break
		}
		independents := map[string]bool{}
		for key, dependents := range dependencies {
			if len(dependents) == 0 {
				// All dependents for the key have been removed
				// Delete this key later in the function
				independents[key] = true
			} else {
				for dependent := range dependents {
					if _, ok := dependencies[dependent]; !ok {
						// This dependent key does not have a key in the dependencies.
						independents[dependent] = true
					}
				}
			}
		}
		// Remove the independent keys from all the entries
		for key, dependents := range dependencies {
			if len(dependents) == 0 {
				// Key already added before. Remove it
				delete(dependencies, key)
			} else {
				for independent := range independents {
					delete(dependents, independent)
				}
			}
		}
		err := callback(independents)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetEnvWithDefault get string value from environment
// variable given by key, or return defVal if env var
// is not set or error occurred
func GetEnvWithDefault(key string, defVal string) string {
	s := os.Getenv(key)
	if s == "" {
		return defVal
	}
	return s
}

// GetEnvIntWithDefault get int value from environment
// variable given by key, or return defVal if env var
// is not set or error occurred
func GetEnvIntWithDefault(key string, defVal int) int {
	s := os.Getenv(key)
	if s == "" {
		return defVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defVal
	}
	return v
}

// MultiValue is to receive multiple values for a flag like --param1 val1 --param1 val2 ..
type MultiValue map[string]bool

// String returns the string representation
func (mv *MultiValue) String() string {
	return strings.Join(mv.Values(), ",")
}

// Set sets the flag
func (mv *MultiValue) Set(value string) error {
	mvv := *mv
	mvv[value] = true
	return nil
}

// Values returns all the values
func (mv *MultiValue) Values() []string {
	mvv := *mv
	keys := make([]string, 0, len(mvv))
	for key := range mvv {
		keys = append(keys, key)
	}
	return keys
}

// MustMarshal panics if the interface cannot be marshalled
func MustMarshal(i interface{}) string {
	bs, err := ConvertToJSON(i)
	if err != nil {
		panic(err)
	}
	return string(bs)
}
