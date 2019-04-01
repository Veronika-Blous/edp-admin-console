package controllers

import (
	"edp-admin-console/service"
	"fmt"
	"github.com/astaxie/beego"
	"net/http"
)

type EdpRestController struct {
	beego.Controller
	EDPTenantService service.EDPTenantService
}

func (this *EdpRestController) GetEDPTenants() {
	resourceAccess := this.Ctx.Input.Session("resource_access").(map[string][]string)
	edpTenants, err := this.EDPTenantService.GetEDPTenants(resourceAccess)
	if err != nil {
		http.Error(this.Ctx.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return
	}

	this.Data["json"] = edpTenants
	this.ServeJSON()
}

func (this *EdpRestController) GetTenantByName() {
	edpTenantName := this.GetString(":name")
	edpTenant, err := this.EDPTenantService.GetTenantByName(edpTenantName)
	if err != nil {
		http.Error(this.Ctx.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return
	}

	if &edpTenantName == nil {
		nonAppMsg := fmt.Sprintf("Please check tenant name. It seems there're not %s tenant.", edpTenantName)
		http.Error(this.Ctx.ResponseWriter, nonAppMsg, http.StatusNotFound)
		return
	}

	this.Data["json"] = edpTenant
	this.ServeJSON()
}
