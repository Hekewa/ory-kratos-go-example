package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	ory "github.com/ory/client-go"
)

func prettyPrint(flow any) {
	flowJson, err := json.MarshalIndent(flow, "", "  ")
	if err != nil {
		return
	}
	fmt.Printf("%s\n", string(flowJson))
}

type App struct {
	ory *ory.APIClient
}

func (a *App) Registration(c echo.Context) error {
	flowId := c.FormValue("flow")
	req := a.ory.FrontendAPI.GetRegistrationFlow(c.Request().Context())
	req = req.Id(flowId)
	req = req.Cookie(c.Request().Header.Get("Cookie"))
	flow, _, err := req.Execute()
	prettyPrint(flow)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return c.Render(http.StatusOK, "kratos-post.html", flow.GetUi())
}

func (a *App) Verification(c echo.Context) error {
	flowId := c.FormValue("flow")
	req := a.ory.FrontendAPI.GetVerificationFlow(c.Request().Context())
	req = req.Id(flowId)
	req = req.Cookie(c.Request().Header.Get("Cookie"))
	flow, _, err := req.Execute()
	prettyPrint(flow)
	if err != nil {
		return c.Redirect(http.StatusInternalServerError, err.Error())
	}
	if flow.Ui.GetMethod() == http.MethodGet {
		return c.Render(http.StatusOK, "kratos-get.html", flow.GetUi())
	}
	return c.Render(http.StatusOK, "kratos-post.html", flow.GetUi())
}

func (a *App) Login(c echo.Context) error {
	flowId := c.FormValue("flow")
	req := a.ory.FrontendAPI.GetLoginFlow(c.Request().Context())
	req = req.Id(flowId)
	req = req.Cookie(c.Request().Header.Get("Cookie"))
	flow, _, err := req.Execute()
	prettyPrint(flow)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return c.Render(http.StatusOK, "kratos-post.html", flow.GetUi())
}

func (a *App) Logout(c echo.Context) error {
	req := a.ory.FrontendAPI.CreateBrowserLogoutFlow(c.Request().Context())
	req = req.Cookie(c.Request().Header.Get("Cookie"))
	flow, _, err := req.Execute()
	prettyPrint(flow)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return c.Redirect(http.StatusTemporaryRedirect, flow.LogoutUrl)
}

func (a *App) Settings(c echo.Context) error {
	flowId := c.FormValue("flow")
	req := a.ory.FrontendAPI.GetSettingsFlow(c.Request().Context())
	req = req.Id(flowId)
	req = req.Cookie(c.Request().Header.Get("Cookie"))
	flow, _, err := req.Execute()
	prettyPrint(flow)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return c.Render(http.StatusOK, "kratos-post.html", flow.GetUi())
}

func (a *App) Dashboard(c echo.Context) error {
	sess, _ := getSession(c.Request().Context())
	prettyPrint(sess)
	return c.Render(http.StatusOK, "dashboard.html", sess.Identity.GetTraits())
}

type TemplateRenderer struct {
	templates *template.Template
}

// Render renders a template document
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func (app *App) sessionMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cookies := c.Request().Header.Get("Cookie")

		session, _, err := app.ory.FrontendAPI.ToSession(c.Request().Context()).Cookie(cookies).Execute()
		if err != nil || !*session.Active {
			return c.Redirect(http.StatusFound, "http://kratos:4433/self-service/login/browser")
		}
		ctx := context.WithValue(c.Request().Context(), "req.session", session)
		c.SetRequest(c.Request().WithContext(ctx))
		return next(c)
	}
}

func getSession(ctx context.Context) (*ory.Session, error) {
	session, ok := ctx.Value("req.session").(*ory.Session)
	if !ok || session == nil {
		return nil, errors.New("session not found in context")
	}
	return session, nil
}

func main() {
	e := echo.New()
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	e.Static("assets", "assets")

	renderer := &TemplateRenderer{
		templates: template.Must(template.ParseGlob("assets/*.html")),
	}
	e.Renderer = renderer

	oryConfig := ory.NewConfiguration()
	oryConfig.Servers = ory.ServerConfigurations{{URL: "http://kratos:4433"}}

	app := &App{
		ory: ory.NewAPIClient(oryConfig),
	}

	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index.html", nil)
	})
	e.GET("/health", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	e.GET("/registration", app.Registration)
	e.GET("/verification", app.Verification)
	e.GET("/login", app.Login)
	e.GET("/logout", app.Logout)
	e.GET("/settings", app.Settings)

	e.GET("/dashboard", app.sessionMiddleware(app.Dashboard))

	e.Logger.Fatal(e.Start(":8001"))
}
