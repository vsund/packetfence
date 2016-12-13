package pfconfigdriver

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"reflect"
	//"github.com/davecgh/go-spew/spew"
)

// Struct that encapsulates the necessary informations to do a query to pfconfig
type Query struct {
	method  string
	ns      string
	payload string
}

// Fetch data from the pfconfig socket for a string payload
// Returns the bytes received from the socket
func FetchSocket(payload string) []byte {
	c, err := net.Dial("unix", "/usr/local/pf/var/run/pfconfig.sock")

	if err != nil {
		panic(err)
	}

	fmt.Fprintf(c, payload)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	buf.ReadFrom(c)
	var length uint32
	binary.Read(&buf, binary.LittleEndian, &length)
	response := make([]byte, length)
	buf.Read(response)
	if uint32(len(response)) != length {
		panic(fmt.Sprintf("Got invalid length response from pfconfig %d expected, received %d", length, len(response)))
	}
	c.Close()
	return response
}

// Lookup the pfconfig metadata for a specific field
// If there is a non-zero value in the field, it will be taken
// Otherwise it will take the value in the val tag of the field
func metadataFromField(param PfconfigObject, fieldName string) string {
	var ov reflect.Value
	switch val := param.(type) {
	case reflect.Value:
		ov = val
	default:
		ov = reflect.ValueOf(param).Elem()
	}
	userVal := reflect.Value(ov.FieldByName(fieldName)).Interface()

	if userVal != "" {
		return userVal.(string)
	}

	ot := ov.Type()
	if field, ok := ot.FieldByName(fieldName); ok {
		val := field.Tag.Get("val")
		if val != "-" {
			return val
		} else {
			panic(fmt.Sprintf("No default value defined for %s on %s. User specified value is required.", fieldName, ot.String()))
		}
	} else {
		panic(fmt.Sprintf("Missing %s for %s", fieldName, ot.String()))
	}
}

// Decode an array of bytes representing a json string into interface
// Panics if there is an error decoding the JSON data
func decodeJsonObject(b []byte, o interface{}) {
	decoder := json.NewDecoder(bytes.NewReader(b))
	for {
		if err := decoder.Decode(&o); err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
	}
}

// Create a pfconfig query given a PfconfigObject
// Will extract the query information from the object and will create the payload accordingly
func createQuery(o PfconfigObject) Query {
	query := Query{}
	query.ns = metadataFromField(o, "PfconfigNS")
	query.method = metadataFromField(o, "PfconfigMethod")
	if query.method == "hash_element" {
		query.ns = query.ns + ";" + metadataFromField(o, "PfconfigHashNS")
	}
	query.payload = fmt.Sprintf(`{"method":"%s", "key":"%s","encoding":"json"}`+"\n", query.method, query.ns)
	return query
}

// Fetch and decode a namespace from pfconfig given a pfconfig compatible struct
// This cannot accept an interface and requires the struct to have been declared to its final type (so not created by the reflection)
func FetchDecodeSocketStruct(o PfconfigObject) {
	FetchDecodeSocket(o, reflect.Value{})
}

// Fetch and decode a namespace from pfconfig given a pfconfig compatible struct
// The proper reflect.Value must be passed to extract the pfconfig metadata from
func FetchDecodeSocketInterface(o PfconfigObject, reflectInfo reflect.Value) {
	FetchDecodeSocket(o, reflectInfo)
}

// Fetch and decode a namespace from pfconfig given a pfconfig compatible struct
// If reflectInfo is a valid reflect.Value, it will be used to extract the pfconfig metadata from it
// This will fetch the json representation from pfconfig and decode it into o
// o must be a pointer to the struct as this should be used by reference
func FetchDecodeSocket(o PfconfigObject, reflectInfo reflect.Value) {
	var queryParam interface{}
	if reflectInfo.IsValid() {
		queryParam = reflectInfo
	} else {
		queryParam = o
	}
	query := createQuery(queryParam)
	jsonResponse := FetchSocket(query.payload)
	if query.method == "keys" {
		if cs, ok := o.(*ConfigSections); ok {
			decodeJsonObject(jsonResponse, &cs.Keys)
		} else {
			panic("Wrong object type for keys. Required ConfigSections")
		}
	} else {
		receiver := &PfconfigElementResponse{}
		decodeJsonObject(jsonResponse, receiver)
		b, _ := receiver.Element.MarshalJSON()
		decodeJsonObject(b, &o)
	}

}