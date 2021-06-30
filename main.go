package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/core"
)

var (
	BarkUrl             = ``                    //通知地址
	Shape               = "VM.Standard.A1.Flex" //配置，不需要修改
	DisplayName         = "uniqueque-oracle-free"
	ocpus       float32 = 3 //你想要的配置,内存=ocpus*6,免费额度最大4
)

func main() {
	config := common.DefaultConfigProvider()
	ctx := context.Background()
	c, err := core.NewComputeClientWithConfigurationProvider(config)
	if err != nil {
		log.Fatalln(err)
	}
	tenancyID, err := config.TenancyOCID()
	if err != nil {
		log.Fatalln(err)
	}
	//获取已存在的实例，判断是升级还是新建
	listInstancesRequest := core.ListInstancesRequest{
		CompartmentId: &tenancyID,
	}
	listInstancesResponse, err := c.ListInstances(ctx, listInstancesRequest)
	instanceId := ``
	for _, instance := range listInstancesResponse.Items {
		// log.Println(*instance.DisplayName, *instance.Shape)
		if *instance.Shape == Shape {
			instanceId = *instance.Id
			break
		}
	}
	log.Println(instanceId)

	if instanceId == "" {
		//todo 自动创建实例
		return
		launchInstanceRequest := core.LaunchInstanceRequest{
			LaunchInstanceDetails: core.LaunchInstanceDetails{
				CompartmentId: &tenancyID,
				DisplayName:   &DisplayName,
				Shape:         &Shape,
				ShapeConfig: &core.LaunchInstanceShapeConfigDetails{
					Ocpus: &ocpus,
				},
				// AvailabilityDomain
			},
		}
		c.LaunchInstance(ctx, launchInstanceRequest)
	}

	//实例id
	getInstanceRequest := core.GetInstanceRequest{InstanceId: &instanceId}
	getInstanceResponse, err := c.GetInstance(ctx, getInstanceRequest)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("当前实例状态", getInstanceResponse.LifecycleState, "当前实例配置", *getInstanceResponse.ShapeConfig.Ocpus, "ocpu")
	update := false
	//是否需要升级
	if *getInstanceResponse.ShapeConfig.Ocpus < ocpus {
		update = true
		log.Println("开始升级实例至", ocpus, "ocpus")
	}
	count := 0
	for update {
		count++
		updateInstanceRequest := core.UpdateInstanceRequest{
			InstanceId: &instanceId,
			UpdateInstanceDetails: core.UpdateInstanceDetails{
				ShapeConfig: &core.UpdateInstanceShapeConfigDetails{Ocpus: &ocpus},
			},
		}
		_, err = c.UpdateInstance(ctx, updateInstanceRequest)
		if err != nil {
			log.Println("第", count, "次升级", err)
			// notify("oracle cloud", err.Error())
			time.Sleep(time.Millisecond * 500)
		} else {
			notify("oracle cloud", "升级成功")
			update = false
		}
	}
	getInstanceResponse, err = c.GetInstance(ctx, getInstanceRequest)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println(getInstanceResponse.LifecycleState, *getInstanceResponse.ShapeConfig.Ocpus)
	//是否需要启动
	run := false
	if getInstanceResponse.LifecycleState == "STOPPED" {
		run = true
	}
	count = 0
	for run {
		count++
		instanceActionRequest := core.InstanceActionRequest{
			InstanceId: &instanceId,
			Action:     core.InstanceActionActionStart,
		}
		instanceActionResponse, err := c.InstanceAction(ctx, instanceActionRequest)
		if err != nil {
			log.Println("第", count, "次启动", err)
			// notify("oracle cloud", err.Error())
			time.Sleep(time.Second * 3)
		} else {
			log.Println(instanceActionResponse.LifecycleState)
			run = false
			notify("oracle cloud", "启动成功")
		}
	}
}

func notify(title, message string) {
	res, err := http.Get(BarkUrl + url.QueryEscape(title) + `/` + url.QueryEscape(message))
	if err != nil {
		log.Println(err)
		return
	}
	defer res.Body.Close()
}
