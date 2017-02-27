package sharings

import (
	"fmt"
	"net/url"

	"github.com/cozy/cozy-stack/pkg/consts"
	"github.com/cozy/cozy-stack/pkg/couchdb"
	"github.com/cozy/cozy-stack/pkg/instance"
	"github.com/cozy/cozy-stack/pkg/jobs"
	"github.com/cozy/cozy-stack/pkg/jobs/workers"
)

// The skeleton of the mail we will send. The values between "{{ }}" will be
// filled through the `mailTemplateValues` structure.
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

// The sharing-dependent information: the recipient's name, the sharer's public
// name, the description of the sharing, and the OAuth query string.
type mailTemplateValues struct {
	RecipientName    string
	SharerPublicName string
	Description      string
	OAuthQueryString string
}

// SendSharingMails will generate the mail containing the details
// regarding this sharing, and will then send it to all the recipients.
func SendSharingMails(instance *instance.Instance, s *Sharing) error {
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
	sharerPublicName, _ := doc.M["public_name"].(string)
	mailValues.SharerPublicName = sharerPublicName

	recipients, err := s.Recipients(instance)
	if err != nil {
		return err
	}
	for _, recipient := range recipients {

		// Generate recipient specific OAuth query string.
		recipientOAuthQueryString, err := generateOAuthQueryString(recipient, s)
		if err != nil {
			return err
		}

		// Augment base values with recipient specific information.
		mailValues.RecipientName = recipient.Email
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
	mailAddresses := []*workers.MailAddress{&workers.MailAddress{Name: r.Email,
		Email: r.Email}}

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
func generateOAuthQueryString(r *Recipient, s *Sharing) (string, error) {
	// In the sharing document the permissions are stored as a
	// `permissions.Set`. We need to convert them in a proper format to be able
	// to incorporate them in the OAuth query string.
	//
	// XXX Optimization: this part could be done outside of this function and
	// also outside of the for loop on the recipients.
	// I found it was clearer to leave it here, at the price of being less
	// optimized.
	permissionsScope, err := s.Permissions.MarshalScopeString()
	if err != nil {
		return "", err
	}

	// We use url.encode to safely escape the query string.
	mapParamQueryString := url.Values{}
	mapParamQueryString["client_id"] = []string{r.Client.ClientID}
	mapParamQueryString["redirect_uri"] = []string{r.Client.RedirectURIs[0]}
	mapParamQueryString["response_type"] = []string{"code"}
	mapParamQueryString["scope"] = []string{permissionsScope}
	mapParamQueryString["sharing_type"] = []string{s.SharingType}
	mapParamQueryString["state"] = []string{s.SharingID}

	paramQueryString := mapParamQueryString.Encode()

	queryString := fmt.Sprintf("%s/sharings/request?%s", r.URL,
		paramQueryString)

	return queryString, nil
}
