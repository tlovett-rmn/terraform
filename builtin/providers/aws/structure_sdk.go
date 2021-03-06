package aws

import (
	"strings"

	"github.com/awslabs/aws-sdk-go/service/ec2"
	"github.com/hashicorp/aws-sdk-go/aws"
	"github.com/hashicorp/aws-sdk-go/gen/elb"
	"github.com/hashicorp/aws-sdk-go/gen/rds"
	"github.com/hashicorp/terraform/helper/schema"
)

// Takes the result of flatmap.Expand for an array of listeners and
// returns ELB API compatible objects
func expandListenersSDK(configured []interface{}) ([]elb.Listener, error) {
	listeners := make([]elb.Listener, 0, len(configured))

	// Loop over our configured listeners and create
	// an array of aws-sdk-go compatabile objects
	for _, lRaw := range configured {
		data := lRaw.(map[string]interface{})

		l := elb.Listener{
			InstancePort:     aws.Integer(data["instance_port"].(int)),
			InstanceProtocol: aws.String(data["instance_protocol"].(string)),
			LoadBalancerPort: aws.Integer(data["lb_port"].(int)),
			Protocol:         aws.String(data["lb_protocol"].(string)),
		}

		if v, ok := data["ssl_certificate_id"]; ok {
			l.SSLCertificateID = aws.String(v.(string))
		}

		listeners = append(listeners, l)
	}

	return listeners, nil
}

// Takes the result of flatmap.Expand for an array of ingress/egress
// security group rules and returns EC2 API compatible objects
func expandIPPermsSDK(
	group ec2.SecurityGroup, configured []interface{}) []*ec2.IPPermission {
	vpc := group.VPCID != nil

	perms := make([]*ec2.IPPermission, len(configured))
	for i, mRaw := range configured {
		var perm ec2.IPPermission
		m := mRaw.(map[string]interface{})

		perm.FromPort = aws.Long(m["from_port"].(int64))
		perm.ToPort = aws.Long(m["to_port"].(int64))
		perm.IPProtocol = aws.String(m["protocol"].(string))

		var groups []string
		if raw, ok := m["security_groups"]; ok {
			list := raw.(*schema.Set).List()
			for _, v := range list {
				groups = append(groups, v.(string))
			}
		}
		if v, ok := m["self"]; ok && v.(bool) {
			if vpc {
				groups = append(groups, *group.GroupID)
			} else {
				groups = append(groups, *group.GroupName)
			}
		}

		if len(groups) > 0 {
			perm.UserIDGroupPairs = make([]*ec2.UserIDGroupPair, len(groups))
			for i, name := range groups {
				ownerId, id := "", name
				if items := strings.Split(id, "/"); len(items) > 1 {
					ownerId, id = items[0], items[1]
				}

				perm.UserIDGroupPairs[i] = &ec2.UserIDGroupPair{
					GroupID: aws.String(id),
					UserID:  aws.String(ownerId),
				}
				if !vpc {
					perm.UserIDGroupPairs[i].GroupID = nil
					perm.UserIDGroupPairs[i].GroupName = aws.String(id)
					perm.UserIDGroupPairs[i].UserID = nil
				}
			}
		}

		if raw, ok := m["cidr_blocks"]; ok {
			list := raw.([]interface{})
			perm.IPRanges = make([]*ec2.IPRange, len(list))
			for i, v := range list {
				perm.IPRanges[i] = &ec2.IPRange{CIDRIP: aws.String(v.(string))}
			}
		}

		perms[i] = &perm
	}

	return perms
}

// Takes the result of flatmap.Expand for an array of parameters and
// returns Parameter API compatible objects
func expandParametersSDK(configured []interface{}) ([]rds.Parameter, error) {
	parameters := make([]rds.Parameter, 0, len(configured))

	// Loop over our configured parameters and create
	// an array of aws-sdk-go compatabile objects
	for _, pRaw := range configured {
		data := pRaw.(map[string]interface{})

		p := rds.Parameter{
			ApplyMethod:    aws.String(data["apply_method"].(string)),
			ParameterName:  aws.String(data["name"].(string)),
			ParameterValue: aws.String(data["value"].(string)),
		}

		parameters = append(parameters, p)
	}

	return parameters, nil
}

// Flattens a health check into something that flatmap.Flatten()
// can handle
func flattenHealthCheckSDK(check *elb.HealthCheck) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, 1)

	chk := make(map[string]interface{})
	chk["unhealthy_threshold"] = *check.UnhealthyThreshold
	chk["healthy_threshold"] = *check.HealthyThreshold
	chk["target"] = *check.Target
	chk["timeout"] = *check.Timeout
	chk["interval"] = *check.Interval

	result = append(result, chk)

	return result
}

// Flattens an array of UserSecurityGroups into a []string
func flattenSecurityGroupsSDK(list []ec2.UserIDGroupPair) []string {
	result := make([]string, 0, len(list))
	for _, g := range list {
		result = append(result, *g.GroupID)
	}
	return result
}

// Flattens an array of Instances into a []string
func flattenInstancesSDK(list []elb.Instance) []string {
	result := make([]string, 0, len(list))
	for _, i := range list {
		result = append(result, *i.InstanceID)
	}
	return result
}

// Expands an array of String Instance IDs into a []Instances
func expandInstanceStringSDK(list []interface{}) []elb.Instance {
	result := make([]elb.Instance, 0, len(list))
	for _, i := range list {
		result = append(result, elb.Instance{aws.String(i.(string))})
	}
	return result
}

// Flattens an array of Listeners into a []map[string]interface{}
func flattenListenersSDK(list []elb.ListenerDescription) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))
	for _, i := range list {
		l := map[string]interface{}{
			"instance_port":     *i.Listener.InstancePort,
			"instance_protocol": strings.ToLower(*i.Listener.InstanceProtocol),
			"lb_port":           *i.Listener.LoadBalancerPort,
			"lb_protocol":       strings.ToLower(*i.Listener.Protocol),
		}
		// SSLCertificateID is optional, and may be nil
		if i.Listener.SSLCertificateID != nil {
			l["ssl_certificate_id"] = *i.Listener.SSLCertificateID
		}
		result = append(result, l)
	}
	return result
}

// Flattens an array of Parameters into a []map[string]interface{}
func flattenParametersSDK(list []rds.Parameter) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))
	for _, i := range list {
		result = append(result, map[string]interface{}{
			"name":  strings.ToLower(*i.ParameterName),
			"value": strings.ToLower(*i.ParameterValue),
		})
	}
	return result
}

// Takes the result of flatmap.Expand for an array of strings
// and returns a []string
func expandStringListSDK(configured []interface{}) []*string {
	vs := make([]*string, 0, len(configured))
	for _, v := range configured {
		vs = append(vs, aws.String(v.(string)))
	}
	return vs
}

//Flattens an array of private ip addresses into a []string, where the elements returned are the IP strings e.g. "192.168.0.0"
func flattenNetworkInterfacesPrivateIPAddessesSDK(dtos []*ec2.NetworkInterfacePrivateIPAddress) []string {
	ips := make([]string, 0, len(dtos))
	for _, v := range dtos {
		ip := *v.PrivateIPAddress
		ips = append(ips, ip)
	}
	return ips
}

//Flattens security group identifiers into a []string, where the elements returned are the GroupIDs
func flattenGroupIdentifiersSDK(dtos []*ec2.GroupIdentifier) []string {
	ids := make([]string, 0, len(dtos))
	for _, v := range dtos {
		group_id := *v.GroupID
		ids = append(ids, group_id)
	}
	return ids
}

//Expands an array of IPs into a ec2 Private IP Address Spec
func expandPrivateIPAddessesSDK(ips []interface{}) []*ec2.PrivateIPAddressSpecification {
	dtos := make([]*ec2.PrivateIPAddressSpecification, 0, len(ips))
	for i, v := range ips {
		new_private_ip := &ec2.PrivateIPAddressSpecification{
			PrivateIPAddress: aws.String(v.(string)),
		}

		new_private_ip.Primary = aws.Boolean(i == 0)

		dtos = append(dtos, new_private_ip)
	}
	return dtos
}

//Flattens network interface attachment into a map[string]interface
func flattenAttachmentSDK(a *ec2.NetworkInterfaceAttachment) map[string]interface{} {
	att := make(map[string]interface{})
	att["instance"] = *a.InstanceID
	att["device_index"] = *a.DeviceIndex
	att["attachment_id"] = *a.AttachmentID
	return att
}
