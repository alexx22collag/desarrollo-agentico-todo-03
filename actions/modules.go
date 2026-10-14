package actions

import (
	"net/http"
	"strings"

	"todo/models"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
)

// ModulesCreate crea un módulo nuevo desde la barra lateral.
func ModulesCreate(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	input := struct {
		Name string `form:"name" json:"name"`
		View string `form:"view" json:"view"`
	}{}
	if err := c.Bind(&input); err != nil {
		return err
	}

	module := &models.Module{Name: strings.TrimSpace(input.Name)}
	verrs, err := tx.ValidateAndCreate(module)
	if err != nil {
		return err
	}

	if verrs.HasAny() {
		view, err := viewOrAll(tx, input.View)
		if err != nil {
			return err
		}
		c.Set("moduleErrors", verrs)
		c.Set("moduleName", module.Name)
		return renderTaskList(c, tx, view, newTaskForView(view), validate.NewErrors(), http.StatusUnprocessableEntity)
	}

	c.Flash().Add("success", "Módulo creado.")
	return c.Redirect(http.StatusSeeOther, "/?module="+module.ID.String())
}
