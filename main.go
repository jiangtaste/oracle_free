package main

//GOOS=linux GOARCH=amd64 go build
//scpto aliyun oracle_free /root/oracle_free/

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/core"
	"github.com/oracle/oci-go-sdk/identity"
)

var (
	BarkUrl         = `` //通知地址
	ocpus   float32 = 4  //你想要的配置,内存=ocpus*6,免费额度最大4

	Shape       = "VM.Standard.A1.Flex" //配置，不需要修改
	DisplayName = "uniqueque-oracle-free"
)
var (
	config    common.ConfigurationProvider
	tenancyID string
)

func main() {
	config = common.DefaultConfigProvider()
	ctx := context.Background()
	c, err := core.NewComputeClientWithConfigurationProvider(config)
	if err != nil {
		log.Fatalln(err)
	}
	tenancyID, err = config.TenancyOCID()
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
	count := 0
	if instanceId == "" {
		log.Println("开始创建实例")
		identityClient, err := identity.NewIdentityClientWithConfigurationProvider(config)
		if err != nil {
			log.Fatalln(err)
		}
		listAvailabilityDomainsRequest := identity.ListAvailabilityDomainsRequest{
			CompartmentId: &tenancyID,
		}
		//获取可用域
		listAvailabilityDomainsResponse, err := identityClient.ListAvailabilityDomains(ctx, listAvailabilityDomainsRequest)
		if err != nil {
			log.Fatalln(err)
		}
		if len(listAvailabilityDomainsResponse.Items) == 0 {
			log.Fatalln("可用域为空")
		}
		// for _, availabilityDomains := range listAvailabilityDomainsResponse.Items {
		// 	log.Println(*availabilityDomains.Name, *availabilityDomains.Id)
		// }
		virtualNetworkClient, err := core.NewVirtualNetworkClientWithConfigurationProvider(config)
		if err != nil {
			log.Fatalln(err)
		}
		//获取网络
		listSubnetsRequest := core.ListSubnetsRequest{
			CompartmentId: &tenancyID,
		}
		listSubnetsResponse, err := virtualNetworkClient.ListSubnets(ctx, listSubnetsRequest)
		if err != nil {
			log.Fatalln(err)
		}
		subnetId := ""
		if len(listSubnetsResponse.Items) == 0 {
			// create a new subnet 没验证过

			// create a new VCN
			createVcnRequest := core.CreateVcnRequest{
				CreateVcnDetails: core.CreateVcnDetails{
					CidrBlock:     common.String("10.0.0.0/16"),
					CompartmentId: &tenancyID,
					DisplayName:   common.String("vcn-unique-oracle-free"),
					DnsLabel:      common.String("vcndns"),
				},
			}
			createVcnResponse, err := virtualNetworkClient.CreateVcn(ctx, createVcnRequest)
			if err != nil {
				log.Fatalln(err)
			}
			createSubnetRequest := core.CreateSubnetRequest{
				CreateSubnetDetails: core.CreateSubnetDetails{
					AvailabilityDomain: listAvailabilityDomainsResponse.Items[0].Name,
					CidrBlock:          common.String("10.0.0.0/24"),
					CompartmentId:      &tenancyID,
					DisplayName:        common.String("subnet-unique-oracle-free"),
					DnsLabel:           common.String("subnetdns1"),
					VcnId:              createVcnResponse.Id,
				},
			}
			createSubnetResponse, err := virtualNetworkClient.CreateSubnet(ctx, createSubnetRequest)
			if err != nil {
				log.Fatalln(err)
			}
			subnetId = *createSubnetResponse.Id
			log.Fatalln("subnet为空")
		} else {
			subnetId = *listSubnetsResponse.Items[0].Id
		}
		// for _, listSubnets := range listSubnetsResponse.Items {
		// 	log.Println(*listSubnets.DisplayName, *listSubnets.VcnId)
		// }
		listImagesRequest := core.ListImagesRequest{
			CompartmentId:   &tenancyID,
			OperatingSystem: common.String("Oracle Linux"),
			Shape:           common.String(Shape),
		}
		listImagesResponse, err := c.ListImages(ctx, listImagesRequest)
		if err != nil {
			log.Fatalln(err)
		}
		imageId := ""
		if len(listImagesResponse.Items) == 0 {
			log.Fatalln("可用镜像为空")
		} else {
			imageId = *listImagesResponse.Items[0].Id
		}
		// for _, image := range listImagesResponse.Items {
		// 	log.Println(*image.DisplayName)
		// }

		for instanceId == "" {
			count++
			//todo add sshkey
			launchInstanceRequest := core.LaunchInstanceRequest{
				LaunchInstanceDetails: core.LaunchInstanceDetails{
					CompartmentId: &tenancyID,
					DisplayName:   &DisplayName,
					Shape:         &Shape,
					ShapeConfig: &core.LaunchInstanceShapeConfigDetails{
						Ocpus: &ocpus,
					},
					AvailabilityDomain: listAvailabilityDomainsResponse.Items[0].Name,
					CreateVnicDetails: &core.CreateVnicDetails{
						SubnetId: &subnetId,
					},
					SourceDetails: core.InstanceSourceViaImageDetails{ImageId: &imageId},
				},
			}
			launchInstanceResponse, err := c.LaunchInstance(ctx, launchInstanceRequest)
			if err != nil {
				if strings.Contains(err.Error(), "error:LimitExceeded.") {
					log.Fatalln("超出免费额度，退出程序")
				}
				log.Println("第", count, "次创建", err)
				time.Sleep(time.Second)
				continue
			}
			notify("oracle cloud", "实例创建成功")
			instanceId = *launchInstanceResponse.Id
			log.Println(instanceId)
		}

	}
	return
	//实例id
	log.Println(instanceId)
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
	count = 0
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
			if strings.Contains(err.Error(), "error:LimitExceeded.") {
				log.Fatalln("超出免费额度，退出程序")
			}
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
	if BarkUrl == "" {
		return
	}
	res, err := http.Get(BarkUrl + url.QueryEscape(title) + `/` + url.QueryEscape(message))
	if err != nil {
		log.Println(err)
		return
	}
	defer res.Body.Close()
}
