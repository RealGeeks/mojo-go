package mojo_test

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/RealGeeks/mojo-go"
)

//
// Contact.MarshalJSON
//

func TestMojoContact_MarshalJSON(t *testing.T) {
	contact := mojo.Contact{
		ID:          "654A4BFB-41B6-4058-B91E-879ECE2C5A0A",
		GroupID:     2,
		Name:        "Jason Polakow",
		Email:       "jason@jp-australia.com",
		Address:     "123 Hana Hwy, Maui",
		MobilePhone: "123-331-1245",
		WorkPhone:   "1238889999",
		HomePhone:   "(891) 234-1213",
	}
	data, err := json.Marshal(contact)

	ok(t, err)
	equals(t, `{`+
		`"api_contact_id":"654A4BFB-41B6-4058-B91E-879ECE2C5A0A",`+
		`"full_name":"Jason Polakow",`+
		`"contactgroup_set":[{"group_id":2}],`+
		`"address":"123 Hana Hwy, Maui",`+
		`"mediainfo_set":[`+
		`{"type":1,"value":"1238889999"},`+
		`{"type":2,"value":"1233311245"},`+
		`{"type":3,"value":"8912341213"},`+
		`{"type":4,"value":"jason@jp-australia.com"}`+
		`]}`, string(data))
}

func TestMojoContact_MarshalJSON_MissingID(t *testing.T) {
	contact := mojo.Contact{
		GroupID: 7,
	}

	_, err := json.Marshal(contact)

	assert(t, err != nil, "should return error")
	equals(t, "json: error calling MarshalJSON for type mojo.Contact: missing required field ID", err.Error())
}

func TestMojoContact_MarshalJSON_MissingGroup(t *testing.T) {
	contact := mojo.Contact{
		ID: "654A4BFB-41B6-4058-B91E-879ECE2C5A0A",
	}

	_, err := json.Marshal(contact)

	assert(t, err != nil, "should return error")
	equals(t, "json: error calling MarshalJSON for type mojo.Contact: missing required field GroupID", err.Error())
}

func TestMojoContact_MarshalJSON_EmptyMediaSet(t *testing.T) {
	contact := mojo.Contact{
		ID:      "654A4BFB-41B6-4058-B91E-879ECE2C5A0A",
		GroupID: 2,
		Name:    "Jason Polakow",
		Address: "123 Hana Hwy, Maui",
	}
	data, err := json.Marshal(contact)

	ok(t, err)
	equals(t, `{`+
		`"api_contact_id":"654A4BFB-41B6-4058-B91E-879ECE2C5A0A",`+
		`"full_name":"Jason Polakow",`+
		`"contactgroup_set":[{"group_id":2}],`+
		`"address":"123 Hana Hwy, Maui"}`, string(data))
}

//
// Mojo.AddContact
//

func TestMojo_AddContact(t *testing.T) {
	var body string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body = readBody(t, r)
		io.WriteString(w, `{"duplicated_api_contact_id": [], "errors": [], "result": [{"api_contact_id": "654A4BFB-41B6-4058-B91E-879ECE2C5A0A", "contact_id": 58}]}`)
	}))
	defer ts.Close()

	client := &mojo.Mojo{
		URL:   ts.URL,
		Token: "5cf3edd8ccc78ea750abdcb9367fb072",
	}
	err := client.AddContact(mojo.Contact{
		ID:      "654A4BFB-41B6-4058-B91E-879ECE2C5A0A",
		GroupID: 2,
		Name:    "Jason Polakow",
	})

	ok(t, err)
	equals(t, `[{`+
		`"api_contact_id":"654A4BFB-41B6-4058-B91E-879ECE2C5A0A",`+
		`"full_name":"Jason Polakow",`+
		`"contactgroup_set":[{"group_id":2}]}]`, body)
}

func TestMojo_AddContact_Duplicate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"duplicated_api_contact_id": ["654A4BFB-41B6-4058-B91E-879ECE2C5A0A"], "errors": ["Duplicated 'api_contact_id': 654A4BFB-41B6-4058-B91E-879ECE2C5A0A"], "result": []}`)
	}))
	defer ts.Close()

	client := &mojo.Mojo{
		URL:   ts.URL,
		Token: "5cf3edd8ccc78ea750abdcb9367fb072",
	}
	err := client.AddContact(mojo.Contact{
		ID:      "654A4BFB-41B6-4058-B91E-879ECE2C5A0A",
		GroupID: 2,
		Name:    "Jason Polakow",
	})

	equals(t, &mojo.ErrDuplicate{ID: "654A4BFB-41B6-4058-B91E-879ECE2C5A0A"}, err)
}

func TestMojo_AddContact_MissingGroup(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"errors": ["All contacts should have the same group_id."], "result": null}`)
	}))
	defer ts.Close()

	client := &mojo.Mojo{
		URL:   ts.URL,
		Token: "5cf3edd8ccc78ea750abdcb9367fb072",
	}
	err := client.AddContact(mojo.Contact{
		ID:      "654A4BFB-41B6-4058-B91E-879ECE2C5A0A",
		GroupID: 2,
		Name:    "Jason Polakow",
	})

	equals(t, &mojo.ErrInvalid{Msg: "All contacts should have the same group_id."}, err)
}

func TestMojo_AddContact_InvalidStatusCode(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		io.WriteString(w, `opssss`)
	}))
	defer ts.Close()

	client := &mojo.Mojo{
		URL:   ts.URL,
		Token: "5cf3edd8ccc78ea750abdcb9367fb072",
	}
	err := client.AddContact(mojo.Contact{
		ID:      "654A4BFB-41B6-4058-B91E-879ECE2C5A0A",
		GroupID: 2,
		Name:    "Jason Polakow",
	})

	assert(t, err != nil, "should return error")
	equals(t, "mojo: invalid status code 500 with body opssss", err.Error())
}

func TestMojo_AddContact_InvalidJSONResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `ops`)
	}))
	defer ts.Close()

	client := &mojo.Mojo{
		URL:   ts.URL,
		Token: "5cf3edd8ccc78ea750abdcb9367fb072",
	}
	err := client.AddContact(mojo.Contact{
		ID:      "654A4BFB-41B6-4058-B91E-879ECE2C5A0A",
		GroupID: 2,
		Name:    "Jason Polakow",
	})

	assert(t, err != nil, "should return error")
	equals(t, "mojo: decoding response body (invalid character 'o' looking for beginning of value)", err.Error())
}

func readBody(t *testing.T, r *http.Request) string {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("Failed to read request body: %s", err)
	}
	return string(body)
}

// assert fails the test if the condition is false.
func assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// ok fails the test if an err is not nil.
func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}
