package slack

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"common/utils"
	"connector-slack/internal/config"
)

//go:embed templates/*/*.tpl
var templateFiles embed.FS

func RenderTemplate(templateName string, templateVars any) (map[string]any, error) {
	tplPath := fmt.Sprintf("templates/%s.tpl", templateName)
	tplBytes, err := templateFiles.ReadFile(tplPath)

	if err != nil {
		return nil, fmt.Errorf("[%s] Invalid or missing template file: %s", err, tplPath)
	}

	tplContent, err := renderTemplateContent(tplPath, string(tplBytes), templateVars)

	if err != nil {
		return nil, fmt.Errorf("[%s] Failed to render embedded template file: %w", tplPath, err)
	}

	var slackBody map[string]any

	if err = json.Unmarshal([]byte(tplContent), &slackBody); err != nil {
		return nil, fmt.Errorf("[%s] Failed to unmarshal content after templating: %w", tplPath, err)
	}

	return slackBody, nil
}

func renderTemplateContent(templateName string, templateContent string, vars any) (string, error) {
	tpl, err := template.New(templateName).Funcs(template.FuncMap{
		"json":         templateJSON,
		"beautifyTime": utils.TimeFromSlack,
		"hasPrefix":    strings.HasPrefix,
		"contains":     strings.Contains,
		"capitalize":   utils.CapitalizeString,
	}).Parse(templateContent)

	if err != nil {
		return "", err
	}

	var out strings.Builder

	if err := tpl.Execute(&out, vars); err != nil {
		return "", err
	}

	return out.String(), nil
}

func templateJSON(value any) string {
	body, err := json.Marshal(value)
	if err != nil {
		return `""`
	}

	return string(body)
}

func wrapReqError(message string, err error) error {
	var clientErr *utils.ClientError
	if ok := AsClientError(err, &clientErr); ok {
		return &config.ClientError{
			Kind:       config.ClientErrorKind(clientErr.Kind),
			Message:    message,
			StatusCode: clientErr.StatusCode,
			Body:       clientErr.Body,
			ParsedJSON: clientErr.ParsedJSON,
			Err:        clientErr.Err,
		}
	}

	return fmt.Errorf("%s: %w", message, err)
}

func AsClientError(err error, target **utils.ClientError) bool {
	if err == nil {
		return false
	}

	clientErr, ok := err.(*utils.ClientError)
	if !ok {
		return false
	}

	*target = clientErr

	return true
}

func asString(value any) string {
	if value == nil {
		return ""
	}

	str, ok := value.(string)
	if !ok {
		return ""
	}

	return str
}

func mustJSON(value any) []byte {
	body, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}

	return body
}
