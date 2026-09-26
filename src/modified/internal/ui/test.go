package ui

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	mi "github.com/ha1tch/minty"
	fe "github.com/ha1tch/seam-ui/internal/formengine"
	testschemas "github.com/ha1tch/seam-ui/internal/formengine/schemas/test"
)

type Test struct {
	ID          int64
	Title       string
	Description string
	AssetName   string
	CreatedAt   string
}

func (h *UIHandler) TestList(w http.ResponseWriter, r *http.Request) {
	t := h.translateFunc()
	ctx := r.Context()

	result, err := h.xoluClient.OQL(ctx, "SELECT TOP 100 * FROM test ORDER BY id DESC")
	var tests []Test
	if err == nil {
		for _, row := range result.Data {
			tests = append(tests, Test{
				ID:          toInt64(row["id"]),
				Title:       toString(row["title"]),
				Description: toString(row["Description"]),
			})
		}
	} else {
		h.logger.Error("failed to list test", "error", err)
	}

	for i := range tests {
		if tests[i].ID > 0 {
			if asset, err := h.xoluClient.Get(ctx, "assets", tests[i].ID); err == nil {
				tests[i].AssetName = toString(asset.Data["name"])
			}
		}
	}

	data := TestListData{Tests: tests}
	h.page.Render(w, r, t("test.list_title"), "/test", TestListPage(data, t))
}

func (h *UIHandler) TestNew(w http.ResponseWriter, r *http.Request) {
	t := h.translateFunc()
	ctx := r.Context()
	csrfToken := h.GetCSRFToken(w, r)

	formEng := &fe.FormEngine{Translator: h.formEngineTranslator()}
	rc := fe.RenderContext{Locale: h.currentLocale(), CSRFToken: csrfToken, Module: "test", Form: "test"}
	formContent := formEng.Render(ctx, testschemas.Schema("create"), nil, nil, rc)

	data := TestFormData{CSRFToken: csrfToken, FormContent: formContent}
	h.page.Render(w, r, t("test.new"), "/test", TestFormPage(data, t))
}

func (h *UIHandler) TestCreate(w http.ResponseWriter, r *http.Request) {
	t := h.translateFunc()
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}
	title := r.FormValue("title")
	description := r.FormValue("description")
	csrfToken := h.GetCSRFToken(w, r)

	errors := map[string]string{}
	if title == "" {
		errors["title"] = t("test.error.title_required")
	}
	if len(errors) > 0 {
		formEng := &fe.FormEngine{Translator: h.formEngineTranslator()}
		rc := fe.RenderContext{Locale: h.currentLocale(), CSRFToken: csrfToken, Module: "test", Form: "test"}
		formContent := formEng.Render(ctx, testschemas.Schema("create"),
			map[string]interface{}{"title": title, "description": description}, errors, rc)
		data := TestFormData{CSRFToken: csrfToken, FormContent: formContent, Errors: errors}
		h.page.Render(w, r, t("test.new"), "/test", TestFormPage(data, t))
		return
	}

	testData := map[string]any{
		"title":       title,
		"description": description,
		"created_at":  time.Now().UTC().Format(time.RFC3339),
	}

	entity, err := h.xoluClient.Create(ctx, "test", testData)
	if err != nil {
		h.logger.Error("failed to create test", "error", err)
		http.Error(w, "Failed to save test", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/test/"+strconv.FormatInt(entity.ID, 10), http.StatusSeeOther)
}

func (h *UIHandler) TestDetail(w http.ResponseWriter, r *http.Request) {
	t := h.translateFunc()
	ctx := r.Context()
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id == 0 {
		h.page.RenderError(w, http.StatusNotFound, t("error.not_found"), t("error.not_found"))
		return
	}

	entity, err := h.xoluClient.Get(ctx, "test", id)
	if err != nil {
		h.page.RenderError(w, http.StatusNotFound, t("error.not_found"), t("error.not_found"))
		return
	}

	test := Test{
		ID:          entity.ID,
		Title:       toString(entity.Data["title"]),
		Description: toString(entity.Data["description"]),
		CreatedAt:   toString(entity.Data["created_at"]),
	}

	data := TestDetailData{Test: test, CSRFToken: h.GetCSRFToken(w, r)}
	h.page.Render(w, r, t("test.detail_title"), "/test", TestDetailPage(data, t))
}

type TestListData struct {
	Tests []Test
}

// IssuesListPage renders the issues list page.
func TestListPage(data TestListData, t func(key string, args ...any) string) mi.H {
	if t == nil {
		t = func(key string, args ...any) string { return key }
	}
	items := make([]SimpleListItem, len(data.Tests))
	for i, iss := range data.Tests {
		items[i] = SimpleListItem{
			Href:     fmt.Sprintf("/test/%d", iss.ID),
			Title:    iss.Title,
			Subtitle: iss.Description,
		}
	}
	return func(b *mi.Builder) mi.Node {
		return b.Div(mi.Class("space-y-6"),
			b.Div(mi.Class("flex items-center justify-between"),
				b.H1(mi.Class("text-2xl font-bold text-gray-900 dark:text-white"), t("test.list_title")),
				b.A(mi.Href("/test/new"), mi.Class("px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 font-medium"),
					t("test.new"),
				),
			),
			SimpleListCard(items, t("test.empty"))(b),
		)
	}
}

type TestFormData struct {
	CSRFToken   string
	FormContent mi.H
	Errors      map[string]string
}

func TestFormPage(data TestFormData, t func(key string, args ...any) string) mi.H {
	if t == nil {
		t = func(key string, args ...any) string { return key }
	}
	return func(b *mi.Builder) mi.Node {
		return b.Div(mi.Class("space-y-6 max-w-2xl"),
			b.Div(mi.Class("flex items-center gap-4"),
				b.A(mi.Href("/test"), mi.Class("text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"),
					b.Span(mi.Class("material-icons"), "arrow_back"),
				),
				b.H1(mi.Class("text-2xl font-bold text-gray-900 dark:text-white"), t("test.new")),
			),
			Card("", func(b *mi.Builder) mi.Node {
				return b.Form(mi.Class("space-y-4"), mi.Method("post"), mi.Action("/test"),
					data.FormContent(b),
					b.Div(mi.Class("flex justify-end gap-3 pt-4"),
						b.A(mi.Href("/test"), mi.Class("px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg font-medium hover:bg-gray-50 dark:hover:bg-gray-700"),
							t("action.cancel"),
						),
						b.Button(mi.Type("submit"), mi.Class("px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 font-medium"),
							t("test.action.create"),
						),
					),
				)
			})(b),
		)
	}
}

type TestDetailData struct {
	Test      Test
	CSRFToken string
}

func TestDetailPage(data TestDetailData, t func(key string, args ...any) string) mi.H {
	if t == nil {
		t = func(key string, args ...any) string { return key }
	}
	iss := data.Test
	return func(b *mi.Builder) mi.Node {
		return b.Div(mi.Class("space-y-6 max-w-2xl"),
			b.Div(mi.Class("flex items-center gap-4"),
				b.A(mi.Href("/test"), mi.Class("text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"),
					b.Span(mi.Class("material-icons"), "arrow_back"),
				),
				b.H1(mi.Class("text-2xl font-bold text-gray-900 dark:text-white"), iss.Title),
			),
			Card("", func(b *mi.Builder) mi.Node {
				var children []interface{}
				children = append(children,
					b.Div(mi.Class("text-sm text-gray-700 dark:text-gray-300"), iss.Description),
				)
				return b.Div(children...)
			})(b),
		)
	}
}
