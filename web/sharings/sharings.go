package sharings

import (
	"net/http"

	"github.com/cozy/cozy-stack/pkg/consts"
	"github.com/cozy/cozy-stack/pkg/couchdb"
	"github.com/cozy/cozy-stack/pkg/sharings"
	"github.com/cozy/cozy-stack/web/jsonapi"
	"github.com/cozy/cozy-stack/web/middlewares"
	"github.com/labstack/echo"
)

// CreateSharing initializes a sharing by creating the associated document
func CreateSharing(c echo.Context) error {
	instance := middlewares.GetInstance(c)

	sharing := new(sharings.Sharing)
	if err := c.Bind(sharing); err != nil {
		return err
	}

	if err := sharings.CheckSharingCreation(instance, sharing); err != nil {
		return wrapErrors(err)
	}

	doc, err := sharings.Create(instance, sharing)
	if err != nil {
		return err
	}

	return jsonapi.Data(c, http.StatusOK, doc, nil)
}

// SendSharingMails sends the mails requests for the provided sharing.
func SendSharingMails(c echo.Context) error {
	// Fetch the instance.
	instance := middlewares.GetInstance(c)

	// Fetch the document id and then the sharing document.
	docID := c.Param("id")
	sharing := &sharings.Sharing{}
	err := couchdb.GetDoc(instance, consts.Sharings, docID, sharing)
	if err != nil {
		// TODO create a new error for non existing document?
		return wrapErrors(err)
	}

	// Send the mails.
	err = sharings.SendSharingMails(instance, sharing)
	if err != nil {
		// TODO create a new error in case the sending failed?
		return wrapErrors(err)
	}

	return nil
}

// Routes sets the routing for the sharing service
func Routes(router *echo.Group) {
	// API Routes
	router.POST("/", CreateSharing)
	router.POST("/:id/sendMails", SendSharingMails)
}

func wrapErrors(err error) error {
	switch err {
	case sharings.ErrBadSharingType:
		return jsonapi.InvalidParameter("sharing_type", err)
	case sharings.ErrRecipientDoesNotExist:
		return jsonapi.NotFound(err)
	}
	return err
}
