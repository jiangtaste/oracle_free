package main

import (
	"context"
	"github.com/oracle/oci-go-sdk/identity"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/core"
)

var (
	BarkUrl                   = ""                    // 通知地址
	sshAuthorizedKeys         = ``                    // 拷贝.ssh/id_rsa.pub的内容

	Shape                = "VM.Standard.A1.Flex" // 配置，不需要修改
	Ocpus        float32 = 2                     // 你想要的配置,内存=Ocpus*6,免费额度最大4
	MaxInstances         = 2                     // 最大实例
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

	if err != nil {
		log.Fatalln(err)
	}

	// 获取实例
	var instanceIds []string
	for _, instance := range listInstancesResponse.Items {
		// log.Println(*instance.DisplayName, *instance.Shape)
		if *instance.Shape == Shape {
			instanceIds = append(instanceIds, *instance.Id)
		}
	}

	log.Println(instanceIds)

	// 创建实例
	if len(instanceIds) == 0 {
		count := 0
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
		instanceName := "uniqueque-oracle-free" + time.Now().Format("-20060102")


		for len(instanceIds) < MaxInstances {
			count++
			launchInstanceRequest := core.LaunchInstanceRequest{
				LaunchInstanceDetails: core.LaunchInstanceDetails{
					CompartmentId: &tenancyID,
					DisplayName:   &instanceName,
					Shape:         &Shape,
					ShapeConfig: &core.LaunchInstanceShapeConfigDetails{
						Ocpus: &Ocpus,
					},
					AvailabilityDomain: listAvailabilityDomainsResponse.Items[0].Name,
					CreateVnicDetails: &core.CreateVnicDetails{
						SubnetId: &subnetId,
					},
					SourceDetails: core.InstanceSourceViaImageDetails{ImageId: &imageId},
					Metadata:      map[string]string{"ssh_authorized_keys": sshAuthorizedKeys},
				},
			}
			launchInstanceResponse, err := c.LaunchInstance(ctx, launchInstanceRequest)
			if err != nil {
				if strings.Contains(err.Error(), "error:LimitExceeded.") {
					log.Fatalln("超出免费额度，退出程序")
				}
				log.Println("第", count, "次创建", err)
				time.Sleep(time.Second * 3)
				continue
			}
			notify("oracle cloud", "实例创建成功")
			instanceIds = append(instanceIds, *launchInstanceResponse.Id)
		}

		log.Println(instanceIds)
	}


	// 更新实例
	for _, instanceId := range instanceIds {
		updateInstance(c, instanceId)
	}

}

func updateInstance(c core.ComputeClient, instanceId string) {

	log.Println(instanceId)

	ctx := context.Background()

	getInstanceRequest := core.GetInstanceRequest{InstanceId: &instanceId}
	getInstanceResponse, err := c.GetInstance(ctx, getInstanceRequest)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("当前实例状态", getInstanceResponse.LifecycleState, "当前实例配置", *getInstanceResponse.ShapeConfig.Ocpus, "ocpus")
	update := false
	//是否需要升级
	if *getInstanceResponse.ShapeConfig.Ocpus < Ocpus {
		update = true
		log.Println("开始升级实例至", Ocpus, "Ocpus")
		notify("oracle cloud", "开始升级instance Id: " + instanceId)
	}
	count := 0
	for update {
		count++
		updateInstanceRequest := core.UpdateInstanceRequest{
			InstanceId: &instanceId,
			UpdateInstanceDetails: core.UpdateInstanceDetails{
				ShapeConfig: &core.UpdateInstanceShapeConfigDetails{Ocpus: &Ocpus},
			},
		}
		_, err = c.UpdateInstance(ctx, updateInstanceRequest)
		if err != nil {
			if strings.Contains(err.Error(), "error:LimitExceeded.") {
				log.Fatalln("超出免费额度，退出程序")
			}
			log.Println("第", count, "次升级", err)
			//notify("oracle cloud", err.Error())
			time.Sleep(time.Minute * 1)
		} else {
			notify("oracle cloud", "升级成功")
			update = false
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
