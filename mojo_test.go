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
		Address:     "123 Hana Hwy",
		City:        "Paia",
		State:       "HI",
		Zip:         "12345",
		MobilePhone: "123-331-1245",
		WorkPhone:   "1238889999",
		HomePhone:   "(891) 234-1213",
		Notes:       []string{"called him today", "should mention new home"},
	}
	data, err := json.Marshal(contact)

	ok(t, err)
	equals(t, `{`+
		`"api_contact_id":"654A4BFB-41B6-4058-B91E-879ECE2C5A0A",`+
		`"full_name":"Jason Polakow",`+
		`"contactgroup_set":[{"group_id":2}],`+
		`"address":"123 Hana Hwy",`+
		`"city":"Paia",`+
		`"state":"HI",`+
		`"zip_code":"12345",`+
		`"contactnote_set":[`+
		`{"type":1,"contents":"called him today"},`+
		`{"type":1,"contents":"should mention new home"}`+
		`],`+
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
	var body []map[string]interface{}
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
	equals(t, []map[string]interface{}{
		map[string]interface{}{
			"api_contact_id":   "654A4BFB-41B6-4058-B91E-879ECE2C5A0A",
			"full_name":        "Jason Polakow",
			"contactgroup_set": []interface{}{map[string]interface{}{"group_id": 2.0}},
		}}, body)
}

func TestMojo_AddContactMultiple(t *testing.T) {
	var body []map[string]interface{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body = readBody(t, r)
		io.WriteString(w, `{"duplicated_api_contact_id": [], "errors": [], "result": [{"api_contact_id": "654A4BFB-41B6-4058-B91E-879ECE2C5A0A", "contact_id": 58}]}`)
	}))
	defer ts.Close()

	client := &mojo.Mojo{
		URL:   ts.URL,
		Token: "5cf3edd8ccc78ea750abdcb9367fb072",
	}
	c1 := mojo.Contact{
		ID:          "654a4bfb41b64058b91e879ece2c5a0a",
		GroupID:     2,
		Name:        "Jason Polakow",
		Address:     "123 Paia, Maui",
		Email:       "jp@jp.com",
		MobilePhone: "808-212-2211",
		WorkPhone:   "808-222-0101",
		HomePhone:   "808-812-8213",
	}
	c2 := mojo.Contact{
		ID:      "2755a963e0e549128c27a9f78ee8afde",
		GroupID: 3,
		Name:    "Amanda",
	}
	err := client.AddContact(c1, c2)

	ok(t, err)
	equals(t, []map[string]interface{}{
		map[string]interface{}{
			"api_contact_id": "654a4bfb41b64058b91e879ece2c5a0a",
			"address":        "123 Paia, Maui",
			"full_name":      "Jason Polakow",
			"mediainfo_set": []interface{}{
				map[string]interface{}{"type": 1.0, "value": "8082220101"},
				map[string]interface{}{"type": 2.0, "value": "8082122211"},
				map[string]interface{}{"type": 3.0, "value": "8088128213"},
				map[string]interface{}{"type": 4.0, "value": "jp@jp.com"}},
			"contactgroup_set": []interface{}{
				map[string]interface{}{"group_id": 2.0}}},
		map[string]interface{}{
			"api_contact_id": "2755a963e0e549128c27a9f78ee8afde",
			"full_name":      "Amanda",
			"contactgroup_set": []interface{}{
				map[string]interface{}{"group_id": 3.0}}},
	}, body)
}

func TestMojo_AddContact_Duplicate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{
			"duplicated_api_contact_id": [
				"a030a3fae0aa57f6bebf368fc4370221",
				"68d480032155501eb2b2ca4c6a053306"
			],
			"errors": ["Duplicated 'api_contact_id': a030a3fae0aa57f6bebf368fc4370221, 68d480032155501eb2b2ca4c6a053306"],
			"result": [{
				"api_contact_id": "f2d4a646cebe53f6b9b3f7b846e11f1d",
				"contact_id": 816
			}, {
				"api_contact_id": "32ed3a5b524758968d676279fdc9aaaf",
				"contact_id": 815
			}]
		}`)
	}))
	defer ts.Close()

	client := &mojo.Mojo{
		URL:   ts.URL,
		Token: "5cf3edd8ccc78ea750abdcb9367fb072",
	}
	err := client.AddContact(
		mojo.Contact{ID: "a030a3fae0aa57f6bebf368fc4370221", GroupID: 2, Name: "Bob"},
		mojo.Contact{ID: "68d480032155501eb2b2ca4c6a053306", GroupID: 2, Name: "Ana"},
	)

	equals(t, &mojo.ErrDuplicate{IDs: []string{"a030a3fae0aa57f6bebf368fc4370221", "68d480032155501eb2b2ca4c6a053306"}}, err)
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

func TestMojo_AddContact_PreviousRequestUnfinished(t *testing.T) {
	// Mojo won't execute contact creation from a new request until the
	// previous one is complete as to prevent duplicates/errors.
	//
	// If a new request comes in while the previous one has not finished
	// they return this error
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"errors": ["Previous request was not finished or was interrupted."], "result": null}`)
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
	equals(t, "mojo: Previous request was not finished or was interrupted.", err.Error())
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

func TestMojo_AddContact_Forbidden(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		io.WriteString(w, `{"detail": "Invalid access_token"}`)
	}))
	defer ts.Close()

	client := &mojo.Mojo{
		URL:   ts.URL,
		Token: "invalid",
	}
	err := client.AddContact(mojo.Contact{
		ID:      "654A4BFB-41B6-4058-B91E-879ECE2C5A0A",
		GroupID: 2,
		Name:    "Jason Polakow",
	})

	equals(t, &mojo.ErrForbidden{Msg: "Invalid access_token"}, err)
}

func TestMojo_AddContact_ForbiddenNotJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		io.WriteString(w, `get out of here`) // unknown json format
	}))
	defer ts.Close()

	client := &mojo.Mojo{
		URL:   ts.URL,
		Token: "invalid",
	}
	err := client.AddContact(mojo.Contact{
		ID:      "654A4BFB-41B6-4058-B91E-879ECE2C5A0A",
		GroupID: 2,
		Name:    "Jason Polakow",
	})

	equals(t, &mojo.ErrForbidden{Msg: `get out of here`}, err)
}

func TestMojo_AddContact_ForbiddenUnknownBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		io.WriteString(w, `{"error": "ops"}`) // unknown json format
	}))
	defer ts.Close()

	client := &mojo.Mojo{
		URL:   ts.URL,
		Token: "invalid",
	}
	err := client.AddContact(mojo.Contact{
		ID:      "654A4BFB-41B6-4058-B91E-879ECE2C5A0A",
		GroupID: 2,
		Name:    "Jason Polakow",
	})

	equals(t, &mojo.ErrForbidden{Msg: `{"error": "ops"}`}, err)
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

func readBody(t *testing.T, r *http.Request) (body []map[string]interface{}) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("Failed to read request body: %s", err)
	}
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("Failed to read request body: %s", err)
	}
	return body
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
