package controllers

import (
	"github.com/astaxie/beego"
)

// Operations about object
type HealthController struct {
	beego.Controller
}

// @Title Capture Order
// @Description Capture order Get
// @Success "i'm alive"
// @router / [get]
func (this *HealthController) Get() {
	this.Data["json"] = map[string]string{"response": "i'm alive!"}
	this.ServeJSON()
}
