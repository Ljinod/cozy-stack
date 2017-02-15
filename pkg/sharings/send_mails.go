package sharings

import (
	"fmt"

	"github.com/cozy/cozy-stack/pkg/consts"
	"github.com/cozy/cozy-stack/pkg/couchdb"
	"github.com/cozy/cozy-stack/pkg/jobs"
	"github.com/cozy/cozy-stack/pkg/jobs/workers"
	"github.com/cozy/cozy-stack/web/middlewares"
	"github.com/labstack/echo"
)

const (
	mailTemplateEn = `
        <hr />
        <h3>Hey {{.RecipientName}}!</h3>
        <p>{{.SharerPublicName}} wants to share {{.Description}} with you! To accept the request copy-paste the following link in the sharing management page of your Cozy. :-)

        <br />

        <a href="{{.OAuthQueryString}}">{{.Description}}</a>
        </p>
    `

	mailTemplateFr = `
        <hr />
        <h3>Bonjour {{.RecipientName}} !</h3>
        <p>{{.SharerPublicName}} veut partager {{.Description}} avec vous ! Pour accepter sa demande, copiez-collez le lien ci-dessous dans la page de gestion des partages de votre Cozy. :-)

        <br />

        <a href="{{.OAuthQueryString}}">{{.Description}}</a>
        </p>
    `
)

type mailTemplateValues struct {
	RecipientName    string
	SharerPublicName string
	Description      string
	OAuthQueryString string
}

// SendSharingRequests will generate the mail containing the details
// regarding this sharing, and will then send it to all the recipients.
func SendSharingRequests(c echo.Context, s *Sharing) error {
	// Extract the instance from the context: we will need it to get to the
	// domain and public name of the Cozy owner as well as to start the mail
	// worker.
	instance := middlewares.GetInstance(c)

	// Generate the context later used by the mail worker to send the mails.
	mailWorkerContext := jobs.NewWorkerContext(instance.Domain)

	// Generate the base values of the mail to send, common to all recipients.
	mailValues := &mailTemplateValues{}
	mailValues.Description = s.Desc
	// Get the Couchdb document describing the instance to get the sharer's
	// public name.
	doc := &couchdb.JSONDoc{}
	err := couchdb.GetDoc(instance, consts.Settings,
		consts.InstanceSettingsID, doc)
	if err != nil {
		return err
	}
	// XXX Do we check the length of the public name to avoid empty fields?
	sharerPublicName, _ := doc.M["public_name"].(string)
	mailValues.SharerPublicName = sharerPublicName

	// In the sharing document the permissions are stored as a
	// `permissions.Set`. We need to convert them in a proper format to be able
	// to incorporate them in the OAuth query string.
	permissionsScope, err := s.Permissions.MarshalScopeString()
	if err != nil {
		return err
	}

	for _, recipient := range s.Recipients {
		// Generate recipient specific OAuth query string.
		recipientOAuthQueryString, err := generateOAuthQueryString(
			recipient, s.SharingID, permissionsScope)
		if err != nil {
			return err
		}

		// Augment base values with recipient specific information.
		mailValues.RecipientName = recipient.Mail
		mailValues.OAuthQueryString = recipientOAuthQueryString

		sharingMessage, err := generateMailMessage(s, recipient, mailValues)
		if err != nil {
			return err
		}

		// We have all that we need to start the mail worker.
		workers.SendMail(mailWorkerContext, sharingMessage)
	}

	return nil
}

// generatemailMessage will extract and compute the relevant information
// from the sharing to generate the mail we will send to the recipient
// specified.
func generateMailMessage(s *Sharing, r *Recipient,
	mailValues *mailTemplateValues) (*jobs.Message, error) {

	// We create the mail parts: its content.
	mailPartEn := workers.MailPart{
		Type: "text/html",
		Body: mailTemplateEn}

	mailPartFr := workers.MailPart{
		Type: "text/html",
		Body: mailTemplateFr}

	mailParts := []*workers.MailPart{&mailPartEn, &mailPartFr}

	// The address of the recipient.
	mailAddresses := []*workers.MailAddress{&workers.MailAddress{Name: r.Mail,
		Email: r.Mail}}

	mailOpts := workers.MailOptions{
		Mode:           "from",
		From:           nil, // Will be filled by the stack.
		To:             mailAddresses,
		Subject:        "New sharing request / Nouvelle demande de partage",
		Dialer:         nil, // Will be filled by the stack.
		Date:           nil, // Will be filled by the stack.
		Parts:          mailParts,
		TemplateValues: mailValues}

	message, err := jobs.NewMessage(jobs.JSONEncoding, mailOpts)
	if err != nil {
		return nil, err
	}

	return message, nil
}

// generateOAuthQueryString takes care of creating a correct OAuth request for
// the given sharing and recipient.
func generateOAuthQueryString(r *Recipient, sharingID string, permissionsScope string) (string, error) {
	queryString := fmt.Sprintf("%s/sharings/request"+ // Url of the recipient.
		"?client_id=%s"+ // client_id of sharer at the recipient.
		"&redirect_uri=%s"+ // redirect_uri specified by sharer.
		"&state=%s"+ // XXX sharing_id or random string?
		"&response_type=code"+
		"&scope=%s", // List of permissions.
		r.URL, r.Client.ClientID, r.Client.RedirectURIs[0], sharingID, permissionsScope)

	return queryString, nil
}
