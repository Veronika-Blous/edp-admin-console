/*
 * Copyright 2019 EPAM Systems.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package repository

import (
	"edp-admin-console/models"
	"github.com/astaxie/beego/orm"
	"strconv"
	"time"
)

type ApplicationEntityRepository struct {
}

func (this ApplicationEntityRepository) GetAllApplications(edpName string) ([]models.Application, error) {
	o := orm.NewOrm()
	var applications []models.Application
	var maps []orm.Params

	_, err := o.Raw("SELECT name,"+
		" min(value) FILTER (WHERE property = 'language') AS language,"+
		" min(value) FILTER (WHERE property = 'build_tool') AS build_tool"+
		" FROM business_entity"+
		" LEFT JOIN be_properties ON business_entity.id = be_properties.be_id"+
		" WHERE tenant=? AND delition = 0"+
		" GROUP BY name", edpName).Values(&maps)

	if err != nil {
		return nil, err
	}

	if maps == nil {
		return nil, nil
	}

	for _, row := range maps {
		applications = append(applications, models.Application{
			Name:      row["name"].(string),
			Language:  row["language"].(string),
			BuildTool: row["build_tool"].(string),
		})
	}
	return applications, nil
}

func (this ApplicationEntityRepository) GetApplication(appName string, edpName string) (*models.ApplicationInfo, error) {
	o := orm.NewOrm()
	var application models.ApplicationInfo
	var maps []orm.Params

	_, err := o.Raw("SELECT business_entity.id,tenant,user_name,available,message,last_time_update,status_name, bs.be_id,name,delition,be_type,"+
		"max(value) FILTER (WHERE property = 'language') AS language,"+
		" max(value) FILTER (WHERE property = 'build_tool') AS build_tool,"+
		" max(value) FILTER (WHERE property = 'framework') AS framework,"+
		" max(value) FILTER (WHERE property = 'strategy') AS strategy,"+
		" max(value) FILTER (WHERE property = 'git_url') AS git_url,"+
		" max(value) FILTER (WHERE property = 'route_site') AS route_site,"+
		" max(value) FILTER (WHERE property = 'route_path') AS route_path, "+
		" max(value) FILTER (WHERE property = 'db_kind') AS db_kind,"+
		" max(value) FILTER (WHERE property = 'db_version') AS db_version,"+
		" max(value) FILTER (WHERE property = 'db_capacity') AS db_capacity,"+
		" max(value) FILTER (WHERE property = 'db_storage') AS db_storage"+
		" FROM business_entity"+
		" LEFT JOIN be_properties ON business_entity.id = be_properties.be_id"+
		" LEFT JOIN be_status as bs ON business_entity.id = bs.be_id "+
		" LEFT JOIN statuses_list as sl ON bs.status = sl.status_id WHERE business_entity.name = ? AND business_entity.tenant=? AND business_entity.delition=0"+
		" GROUP BY business_entity.id,tenant,user_name,available,message,last_time_update,status_name, bs.be_id,name,delition,be_type order by last_time_update DESC limit(1)", appName, edpName).Values(&maps)

	if err != nil {
		return nil, err
	}

	if maps == nil {
		return nil, nil
	}

	for _, row := range maps {
		application = models.ApplicationInfo{
			Name:      row["name"].(string),
			Tenant:    row["tenant"].(string),
			Type:      row["be_type"].(string),
			Status:    row["status_name"].(string),
			Language:  row["language"].(string),
			BuildTool: row["build_tool"].(string),
			Framework: row["framework"].(string),
			Strategy:  row["strategy"].(string),
			UserName:  row["user_name"].(string),
			Message:   row["message"].(string),
		}

		application.DelitionTime = formatUnixTimestamp(row["delition"].(string))
		application.LastTimeUpdate = formatUnixTimestamp(row["last_time_update"].(string))

		available, _ := strconv.ParseBool(row["available"].(string))
		application.Available = available

		if row["git_url"] != nil {
			application.GitUrl = row["git_url"].(string)
		}

		if row["route_site"] != nil {
			application.RouteSite = row["route_site"].(string)
		}

		if row["route_path"] != nil {
			application.RoutePath = row["route_path"].(string)
		}

		if row["db_kind"] != nil && row["db_version"] != nil && row["db_capacity"] != nil && row["db_storage"] != nil {
			application.DbKind = row["db_kind"].(string)
			application.DbVersion = row["db_version"].(string)
			application.DbCapacity = row["db_capacity"].(string)
			application.DbStorage = row["db_storage"].(string)
		}
	}
	return &application, nil
}

func formatUnixTimestamp(timestamp string) string {
	tempTime, _ := strconv.ParseInt(timestamp, 10, 64)
	return time.Unix(tempTime, 0).Format("2006-01-02 15:04:05")
}
