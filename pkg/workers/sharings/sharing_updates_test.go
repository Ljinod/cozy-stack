package sharings

import (
	"testing"

	"github.com/cozy/cozy-stack/pkg/consts"
	"github.com/cozy/cozy-stack/pkg/couchdb"
	"github.com/cozy/cozy-stack/pkg/instance"
	"github.com/cozy/cozy-stack/pkg/jobs"
	"github.com/cozy/cozy-stack/pkg/permissions"
	"github.com/stretchr/testify/assert"
)

func createDoc(t *testing.T, docType string, params map[string]interface{}) couchdb.JSONDoc {
	// map are references, so beware to remove previous set values
	delete(params, "_id")
	delete(params, "_rev")
	doc := couchdb.JSONDoc{
		Type: docType,
		M:    params,
	}
	err := couchdb.CreateDoc(in, &doc)
	assert.NoError(t, err)

	return doc
}

func createEvent(t *testing.T, doc couchdb.JSONDoc, sharingID, eventType string) *TriggerEvent {
	msg := &SharingMessage{
		SharingID: sharingID,
		Rule: permissions.Rule{
			Description: "randomdesc",
			Selector:    "",
			Type:        doc.DocType(),
			Values:      []string{},
		},
	}
	event := &TriggerEvent{
		Event:   &EventDoc{Type: eventType, Doc: &doc},
		Message: msg,
	}
	return event
}

func createSharingDocument(i *instance.Instance, doctype string, owner bool, verbs []string) (*couchdb.JSONDoc, error) {
	sDoc := &couchdb.JSONDoc{
		Type: "io.cozy.sharings",
		M: map[string]interface{}{
			"type":  consts.OneShotSharing,
			"owner": owner,
			"desc":  "randomdesc",
			"permissions": []map[string]interface{}{
				{
					"type":     doctype,
					"selector": "referenced_by",
					"values":   []interface{}{"io.cozy.events/random"},
					"verbs":    verbs,
				},
			},
		},
	}

	err := couchdb.CreateDoc(i, sDoc)
	return sDoc, err
}

func TestSharingUpdatesNoSharing(t *testing.T) {
	doc := createDoc(t, testDocType, map[string]interface{}{"test": "test"})
	defer func() {
		couchdb.DeleteDoc(in, doc)
	}()
	event := createEvent(t, doc, "", "CREATED")

	msg, err := jobs.NewMessage(jobs.JSONEncoding, event)
	assert.NoError(t, err)

	err = SharingUpdates(jobs.NewWorkerContext(domainSharer, "123"), msg)
	assert.Error(t, err)
	assert.Equal(t, "Sharing does not exist", err.Error())

}

func TestSharingUpdatesBadSharing(t *testing.T) {
	params := map[string]interface{}{
		"sharing_id": "mysharona",
	}
	doc := createDoc(t, testDocType, params)
	sharingDoc := createDoc(t, consts.Sharings, params)
	defer func() {
		couchdb.DeleteDoc(in, doc)
		couchdb.DeleteDoc(in, sharingDoc)
	}()

	event := createEvent(t, doc, "badsharingid", "")

	msg, err := jobs.NewMessage(jobs.JSONEncoding, event)
	assert.NoError(t, err)

	err = SharingUpdates(jobs.NewWorkerContext(domainSharer, "123"), msg)
	assert.Error(t, err)
	assert.Equal(t, ErrSharingDoesNotExist, err)

}

func TestSharingUpdatesTooManySharing(t *testing.T) {
	params := map[string]interface{}{
		"sharing_id": "mysharona",
	}
	doc := createDoc(t, testDocType, params)
	sharingDoc := createDoc(t, consts.Sharings, params)
	sharingDoc2 := createDoc(t, consts.Sharings, params)
	defer func() {
		couchdb.DeleteDoc(in, doc)
		couchdb.DeleteDoc(in, sharingDoc)
		couchdb.DeleteDoc(in, sharingDoc2)

	}()
	sharingID := doc.M["sharing_id"].(string)

	event := createEvent(t, doc, sharingID, "UPDATED")

	msg, err := jobs.NewMessage(jobs.JSONEncoding, event)
	assert.NoError(t, err)

	err = SharingUpdates(jobs.NewWorkerContext(domainSharer, "123"), msg)
	assert.Error(t, err)
	assert.Equal(t, ErrSharingIDNotUnique, err)
}

func TestSharingUpdatesBadSharingType(t *testing.T) {
	params := map[string]interface{}{
		"sharing_id":   "mysharona.badtype",
		"sharing_type": consts.OneShotSharing,
	}
	doc := createDoc(t, testDocType, params)
	sharingDoc := createDoc(t, consts.Sharings, params)
	defer func() {
		couchdb.DeleteDoc(in, doc)
		couchdb.DeleteDoc(in, sharingDoc)
	}()
	sharingID := sharingDoc.M["sharing_id"].(string)
	event := createEvent(t, doc, sharingID, "UPDATED")

	msg, err := jobs.NewMessage(jobs.JSONEncoding, event)
	assert.NoError(t, err)

	err = SharingUpdates(jobs.NewWorkerContext(domainSharer, "123"), msg)
	assert.Error(t, err)
	assert.Equal(t, ErrDocumentNotLegitimate, err)
}

func TestSharingUpdatesNoRecipient(t *testing.T) {
	params := map[string]interface{}{
		"test": "testy",
	}
	doc := createDoc(t, testDocType, params)

	sharingParams := map[string]interface{}{
		"sharing_id": "mysharona.norecipient",
	}
	r := permissions.Rule{
		Values: []string{doc.ID()},
	}
	perm := permissions.Set{r}
	sharingParams["permissions"] = perm

	sharingDoc := createDoc(t, consts.Sharings, sharingParams)
	defer func() {
		couchdb.DeleteDoc(in, doc)
		couchdb.DeleteDoc(in, sharingDoc)
	}()
	sharingID := sharingDoc.M["sharing_id"].(string)
	event := createEvent(t, doc, sharingID, "CREATED")

	msg, err := jobs.NewMessage(jobs.JSONEncoding, event)
	assert.NoError(t, err)

	err = SharingUpdates(jobs.NewWorkerContext(domainSharer, "123"), msg)
	assert.NoError(t, err)
}

func TestSharingUpdatesBadRecipient(t *testing.T) {
	params := map[string]interface{}{
		"test": "testy",
	}
	doc := createDoc(t, testDocType, params)

	sharingParams := map[string]interface{}{
		"sharing_id": "mysharona.badrecipient",
	}
	r := permissions.Rule{
		Values: []string{doc.ID()},
	}
	perm := permissions.Set{r}
	sharingParams["permissions"] = perm

	sharingDoc := createDoc(t, consts.Sharings, sharingParams)
	defer func() {
		couchdb.DeleteDoc(in, doc)
		couchdb.DeleteDoc(in, sharingDoc)
	}()
	sharingID := sharingDoc.M["sharing_id"].(string)
	event := createEvent(t, doc, sharingID, "CREATED")

	msg, err := jobs.NewMessage(jobs.JSONEncoding, event)
	assert.NoError(t, err)

	err = SharingUpdates(jobs.NewWorkerContext(domainSharer, "123"), msg)
	assert.NoError(t, err)
}

func TestIsDocumentStillShared(t *testing.T) {
	sharedRef := []couchdb.DocReference{
		couchdb.DocReference{Type: "io.cozy.events", ID: "random"},
	}

	optsNotShared := SendOptions{
		Selector: consts.SelectorReferencedBy,
		Values:   []string{"io.cozy.events/static"},
	}
	assert.False(t, isDocumentStillShared(&optsNotShared, sharedRef))

	optsShared := SendOptions{
		Selector: consts.SelectorReferencedBy,
		Values:   []string{"io.cozy.events/random"},
	}
	assert.True(t, isDocumentStillShared(&optsShared, sharedRef))

	optsNotShared = SendOptions{
		Values: []string{"123"},
		DocID:  "456",
	}
	assert.False(t, isDocumentStillShared(&optsNotShared, sharedRef))

}
