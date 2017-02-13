package sharings

import (
	"net/http"

	"github.com/cozy/cozy-stack/pkg/consts"
	"github.com/cozy/cozy-stack/pkg/instance"
	"github.com/cozy/cozy-stack/pkg/sharings"
	"github.com/cozy/cozy-stack/web/jsonapi"
	"github.com/cozy/cozy-stack/web/middlewares"
	"github.com/labstack/echo"
)

func extractRecipients(c echo.Context, instance *instance.Instance) error {

	//TODO: extract the ResourceIdentifier from JSON and get the recipients doc

	return nil
}

func checkStruct(c echo.Context, sharing *sharings.Sharing) error {

	//check sharing type field
	sharingType := sharing.SharingType
	if sharingType != consts.OneShotSharing &&
		sharingType != consts.MasterSlaveSharing &&
		sharingType != consts.MasterMasterSharing {
		err := sharings.ErrBadSharingType
		return jsonapi.InvalidParameter("sharing_type", err)
	}

	//TODO: check other fiels?

	return nil
}

// CreateSharing initializes a sharing by creating a document
func CreateSharing(c echo.Context) error {

	instance := middlewares.GetInstance(c)

	//check recipients
	if err := extractRecipients(c, instance); err != nil {
		return err
	}

	//get the sharing
	sharing := new(sharings.Sharing)
	if err := c.Bind(sharing); err != nil {
		return err
	}

	//check sharing fields
	if err := checkStruct(c, sharing); err != nil {
		return err
	}

	// create document
	doc, err := sharings.Create(instance, sharing)
	if err != nil {
		return err
	}

	return jsonapi.Data(c, http.StatusOK, doc, nil)
}

// Routes sets the routing for the sharing service
func Routes(router *echo.Group) {
	// API Routes
	router.POST("/", CreateSharing)
}

// wrapErrors returns a formatted error
func wrapErrors(err error) error {
	switch err {
	case sharings.ErrBadSharingType:
		return jsonapi.InvalidParameter("sharing_type", err)
	case sharings.ErrRecipientDoesNotExist:
		return jsonapi.NotFound(err)
	}
	return err
}
