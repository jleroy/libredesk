package template

import (
	"fmt"
	"html/template"
	"os"
	"regexp"
	"strings"
	"testing"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/volatiletech/null/v9"
)

var notificationSeedPattern = regexp.MustCompile(`(?s)VALUES\s*\(\s*'email_notification'::template_type,\s*'((?:''|[^'])*)'\s*,\s*false,\s*'([^']+)'`)

func TestRenderNotificationEmailUsesSystemLayout(t *testing.T) {
	m := emailTemplateManager(t)
	var header strings.Builder
	if err := m.tpls.ExecuteTemplate(&header, "header", nil /** data **/); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Customer A", "Customer B"} {
		t.Run(name, func(t *testing.T) {
			data := map[string]any{
				"Author":  map[string]any{"FirstName": name, "FullName": name},
				"Message": map[string]any{"Content": `<script>alert("test")</script> {{ .Author.FirstName }}`},
			}
			got, err := m.RenderNotificationEmail(`<p>{{ .Author.FullName }} replied.</p><blockquote>{{ .Message.Content }}</blockquote>`, data)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(got, name+" replied.") {
				t.Fatal("notification lost the reply author")
			}
			if strings.Count(got, name) != 1 || strings.Contains(got, "Best regards") {
				t.Fatal("notification added a customer signature")
			}
			if strings.Contains(got, "<script>") || !strings.Contains(got, "{{ .Author.FirstName }}") {
				t.Fatal("message content was interpreted as HTML or template code")
			}
			if !strings.Contains(got, "<html>") || !strings.Contains(got, "</html>") || !strings.Contains(got, "support.example.com/images/logo.png") {
				t.Fatal("notification lost the system layout")
			}
		})
	}
}

func TestRenderNotificationEmailUsesCurrentFunctions(t *testing.T) {
	layout := template.Must(template.New("emails").Parse(`{{ define "header" }}<html><body>{{ end }}{{ define "footer" }}</body></html>{{ end }}`))
	m := &Manager{}
	for _, rootURL := range []string{"https://support.example.com", "https://updated.example.com"} {
		funcs := template.FuncMap{"RootURL": func() string { return rootURL }}
		if err := m.Reload(nil /** webTpls **/, layout, funcs); err != nil {
			t.Fatal(err)
		}
		got, err := m.RenderNotificationEmail(`<a href="{{ RootURL }}">View conversation</a>`, nil /** data **/)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got, `href="`+rootURL+`"`) {
			t.Fatal("notification did not use the current template functions")
		}
	}
}

func TestBuildConversationData(t *testing.T) {
	conversation := cmodels.Conversation{
		ReferenceNumber: "C-42",
		Subject:         null.StringFrom("Account <question>"),
		Priority:        null.StringFrom("High"),
		UUID:            "conversation-a",
		InboxName:       "Support",
		InboxChannel:    "email",
		Contact: cmodels.ConversationContact{
			FirstName: "Customer",
			LastName:  "A",
			Email:     null.StringFrom("customer@example.com"),
		},
	}
	customer := umodels.User{FirstName: "Customer", LastName: "A", Email: null.StringFrom("customer@example.com")}
	agent := umodels.User{FirstName: "Agent", LastName: "B", Email: null.StringFrom("agent@example.com")}
	content := template.Must(template.New("content").Option("missingkey=error").Parse(
		`{{ .Conversation.ReferenceNumber }}|{{ .Conversation.Subject }}|{{ .Conversation.Priority }}|{{ .Conversation.UUID }}|{{ .Inbox.Name }}|{{ .Inbox.Channel }}|{{ .Contact.FirstName }}|{{ .Contact.LastName }}|{{ .Contact.FullName }}|{{ .Contact.Email }}|{{ .Recipient.FirstName }}|{{ .Recipient.LastName }}|{{ .Recipient.FullName }}|{{ .Recipient.Email }}|{{ .Author.FirstName }}|{{ .Author.LastName }}|{{ .Author.FullName }}|{{ .Author.Email }}|{{ .IsAIComposed }}`))
	for _, tt := range []struct {
		name      string
		recipient umodels.User
		author    umodels.User
		want      string
	}{
		{
			name:      "customer reply",
			recipient: customer,
			author:    agent,
			want:      "Customer|A|Customer A|customer@example.com|Agent|B|Agent B|agent@example.com|false",
		},
		{
			name:      "incoming reply alert",
			recipient: agent,
			author:    customer,
			want:      "Agent|B|Agent B|agent@example.com|Customer|A|Customer A|customer@example.com|false",
		},
		{
			name:      "automatic alert",
			recipient: agent,
			want:      "Agent|B|Agent B|agent@example.com||| ||false",
		},
		{
			name:      "automated customer reply",
			recipient: customer,
			author:    umodels.User{FirstName: "System", Email: null.StringFrom(umodels.SystemUserEmail)},
			want:      "Customer|A|Customer A|customer@example.com|||||false",
		},
		{
			name:      "AI reply",
			recipient: customer,
			author:    umodels.User{FirstName: "Assistant", Type: umodels.UserTypeAIAssistant},
			want:      "Customer|A|Customer A|customer@example.com|Assistant||Assistant ||true",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			data := BuildConversationData(conversation, tt.recipient, tt.author)
			var rendered strings.Builder
			if err := content.Execute(&rendered, data); err != nil {
				t.Fatal(err)
			}
			want := "C-42|Account &lt;question&gt;|High|conversation-a|Support|email|Customer|A|Customer A|customer@example.com|" + tt.want
			if rendered.String() != want {
				t.Fatalf("rendered = %q, want %q", rendered.String(), want)
			}
		})
	}
}

func TestDefaultNotificationTemplatesRenderForRecipient(t *testing.T) {
	schema, err := os.ReadFile("../../schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	bodies := make(map[string]string)
	for _, match := range notificationSeedPattern.FindAllStringSubmatch(string(schema), -1) {
		bodies[match[2]] = strings.ReplaceAll(match[1], "''", "'")
	}
	m := emailTemplateManager(t)
	conversation := cmodels.Conversation{
		UUID: "conversation-a", ReferenceNumber: "C-42", Subject: null.StringFrom("Account <question>"),
		InboxName: "Support", InboxChannel: "email",
		Contact: cmodels.ConversationContact{FirstName: "Customer", LastName: "A"},
	}
	recipient := umodels.User{FirstName: "Agent", LastName: "B"}
	for _, name := range []string{
		TmplConversationAssigned, TmplNewReply, TmplNewReplyParticipating, TmplConversationReopened,
		TmplMentioned, TmplSLABreachWarning, TmplSLABreached,
	} {
		t.Run(name, func(t *testing.T) {
			body, ok := bodies[name]
			if !ok {
				t.Fatalf("missing default template %q", name)
			}
			data := BuildConversationData(conversation, recipient, umodels.User{FirstName: "Customer", LastName: "A"})
			data["MentionedBy"] = data["Author"]
			data["Message"] = map[string]any{"UUID": "message-a", "Content": "Latest <reply>"}
			data["SLA"] = map[string]any{"Metric": "First response", "DueIn": "5 minutes", "OverdueBy": "2 minutes"}
			got, err := m.RenderNotificationEmail(body, data)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Count(got, "Hi Agent,") != 1 || strings.Contains(got, "Hi Customer,") {
				t.Fatalf("notification greeting used the wrong person: %s", got)
			}
			if !strings.Contains(got, "C-42") || !strings.Contains(got, "/conversation/conversation-a") {
				t.Fatal("notification lost its conversation reference or link")
			}
			if strings.Contains(body, ".Message.Content") && !strings.Contains(got, "Latest &lt;reply&gt;") {
				t.Fatal("notification lost or interpreted the reply content")
			}
		})
	}
}

func TestNotificationRenderingKeepsRecipientsAndSystemEmailsSeparate(t *testing.T) {
	m := emailTemplateManager(t)
	welcomeData := map[string]any{"Email": "agent@example.com", "ResetToken": "test-token"}
	welcome, err := m.RenderInMemoryTemplate(TmplWelcome, welcomeData)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("parallel notifications", func(t *testing.T) {
		for index := range 8 {
			t.Run(fmt.Sprintf("recipient %d", index), func(t *testing.T) {
				t.Parallel()
				recipient := fmt.Sprintf("Agent %d", index)
				body := fmt.Sprintf(`<p>Notice %d: {{ .Recipient }}</p>`, index)
				want := fmt.Sprintf("Notice %d: %s", index, recipient)
				for range 3 {
					got, err := m.RenderNotificationEmail(body, map[string]any{"Recipient": recipient})
					if err != nil {
						t.Fatal(err)
					}
					if !strings.Contains(got, want) || strings.Contains(got, "agent@example.com") || strings.Count(got, "Notice ") != 1 {
						t.Fatalf("notification contained another template or recipient: %s", got)
					}
				}
			})
		}
	})
	after, err := m.RenderInMemoryTemplate(TmplWelcome, welcomeData)
	if err != nil {
		t.Fatal(err)
	}
	if after != welcome || !strings.Contains(after, "test-token") {
		t.Fatal("notification rendering changed the welcome email")
	}
}

func TestBrokenNotificationTemplateDoesNotAffectLaterEmails(t *testing.T) {
	m := emailTemplateManager(t)
	data := map[string]any{"Recipient": map[string]any{"FirstName": "Agent"}}
	for _, body := range []string{"{{", "{{ .Recipient.FirstName.Unknown }}"} {
		t.Run(body, func(t *testing.T) {
			if _, err := m.RenderNotificationEmail(body, data); err == nil {
				t.Fatal("broken notification template rendered without an error")
			}
			got, err := m.RenderNotificationEmail("<p>Hi {{ .Recipient.FirstName }},</p>", data)
			if err != nil || !strings.Contains(got, "Hi Agent,") {
				t.Fatalf("valid notification failed after a broken template: %v", err)
			}
		})
	}
}

func emailTemplateManager(t *testing.T) *Manager {
	t.Helper()
	funcs := template.FuncMap{
		"RootURL":  func() string { return "https://support.example.com" },
		"SiteName": func() string { return "Support" },
		"LogoURL":  func() string { return "" },
	}
	layout, err := template.New("emails").Option("missingkey=error").Funcs(funcs).ParseFiles(
		"../../static/email-templates/base.html", "../../static/email-templates/welcome.html", "../../static/email-templates/reset-password.html")
	if err != nil {
		t.Fatal(err)
	}
	m := &Manager{}
	if err := m.Reload(nil /** webTpls **/, layout, funcs); err != nil {
		t.Fatal(err)
	}
	return m
}
