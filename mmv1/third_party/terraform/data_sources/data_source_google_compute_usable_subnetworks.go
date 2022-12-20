package google

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"google.golang.org/api/compute/v1"
)

func dataSourceGoogleComputeUsableSubnetworks() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceGoogleComputeUsableSubnetworksRead,

		Schema: map[string]*schema.Schema{
			"subnetworks": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"subnetwork": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: `The URL of the subnetwork.`,
						},
						"network": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: `The URL of the network`,
						},
						"ip_cidr_range": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: `The range of internal addresses that are owned by this subnetwork.`,
						},
						"secondary_ip_ranges": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"range_name": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: `The name associated with this subnetwork secondary range, used when adding an alias IP range to a VM instance`,
									},
									"ip_cidr_range": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: `The range of IP addresses belonging to this subnetwork secondary range.`,
									},
								},
							},
						},
						"stack_type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: `The stack type for the subnet.`,
						},
						"ipv6_access_type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: `The access type of IPv6 address this subnet holds.`,
						},
						"purpose": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: `The purpose of the resource.`,
						},
						"role": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: `The role of subnetwork.`,
						},
						"external_ipv6_prefix": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: `The external IPv6 address range that is assigned to this subnetwork.`,
						},
						"internal_ipv6_prefix": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: `The internal IPv6 address range that is assigned to this subnetwork.`,
						},
					},
				},
				Description: `A list of usable subnetwork URLs.`,
			},

			"filter": {
				Type: schema.TypeString,
				Description: `Filter sets the optional parameter "filter": A filter expression that
filters resources listed in the response. The expression must specify
the field name, an operator, and the value that you want to use for
filtering. The value must be a string, a number, or a boolean. The
operator must be either "=", "!=", ">", "<", "<=", ">=" or ":". For
example, if you are filtering Compute Engine instances, you can
exclude instances named "example-instance" by specifying "name !=
example-instance". The ":" operator can be used with string fields to
match substrings. For non-string fields it is equivalent to the "="
operator. The ":*" comparison can be used to test whether a key has
been defined. For example, to find all objects with "owner" label
use: """ labels.owner:* """ You can also filter nested fields. For
example, you could specify "scheduling.automaticRestart = false" to
include instances only if they are not scheduled for automatic
restarts. You can use filtering on nested fields to filter based on
resource labels. To filter on multiple expressions, provide each
separate expression within parentheses. For example: """
(scheduling.automaticRestart = true) (cpuPlatform = "Intel Skylake")
""" By default, each expression is an "AND" expression. However, you
can include "AND" and "OR" expressions explicitly. For example: """
(cpuPlatform = "Intel Skylake") OR (cpuPlatform = "Intel Broadwell")
AND (scheduling.automaticRestart = true) """`,
				Optional: true,
			},

			"project": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: `The google project in which usable subnetworks are listed. Defaults to provider's configuration if missing.`,
			},
		},
	}
}

func dataSourceGoogleComputeUsableSubnetworksRead(context context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*Config)
	userAgent, err := generateUserAgentString(d, config.userAgent)
	if err != nil {
		return diag.FromErr(err)
	}

	project, err := getProject(d, config)
	if err != nil {
		return diag.FromErr(err)
	}

	allUsableSubnetworks := make([]map[string]interface{}, 0)

	req := config.NewComputeClient(userAgent).Subnetworks.ListUsable(project)
	if filter, ok := d.GetOk("filter"); ok {
		req = req.Filter(filter.(string))
	}
	err = req.Pages(context, func(subnetworks *compute.UsableSubnetworksAggregatedList) error {
		for _, item := range subnetworks.Items {
			allUsableSubnetworks = append(allUsableSubnetworks, generateTfSubnetwork(item))
		}
		return nil
	})

	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("subnetworks", allUsableSubnetworks); err != nil {
		return diag.FromErr(fmt.Errorf("error setting subnetworks: %s", err))
	}

	if err := d.Set("project", project); err != nil {
		return diag.FromErr(fmt.Errorf("error setting project: %s", err))
	}
	d.SetId(computeUsableSubnetworksListId(project, d))
	return nil
}

func generateTfSubnetwork(usableSubnetwork *compute.UsableSubnetwork) map[string]interface{} {
	return map[string]interface{}{
		"subnetwork":           usableSubnetwork.Subnetwork,
		"network":              usableSubnetwork.Network,
		"ip_cidr_range":        usableSubnetwork.IpCidrRange,
		"secondary_ip_ranges":  generateTfSecondaryIpRanges(usableSubnetwork.SecondaryIpRanges),
		"stack_type":           usableSubnetwork.StackType,
		"ipv6_access_type":     usableSubnetwork.Ipv6AccessType,
		"purpose":              usableSubnetwork.Purpose,
		"role":                 usableSubnetwork.Role,
		"external_ipv6_prefix": usableSubnetwork.ExternalIpv6Prefix,
		"internal_ipv6_prefix": usableSubnetwork.InternalIpv6Prefix,
	}
}

func generateTfSecondaryIpRanges(secondaryIpRanges []*compute.UsableSubnetworkSecondaryRange) []map[string]interface{} {

	allSecondaryIpRanges := make([]map[string]interface{}, 0)

	for _, secIpRange := range secondaryIpRanges {
		allSecondaryIpRanges = append(allSecondaryIpRanges, generateTfSecondaryIpRange(secIpRange))
	}

	return allSecondaryIpRanges
}

func generateTfSecondaryIpRange(secondaryIpRange *compute.UsableSubnetworkSecondaryRange) map[string]interface{} {
	return map[string]interface{}{
		"range_name":    secondaryIpRange.RangeName,
		"ip_cidr_range": secondaryIpRange.IpCidrRange,
	}
}

func computeUsableSubnetworksListId(project string, d *schema.ResourceData) string {
	filter := "ALL"
	if subfilter, ok := d.GetOk("filter"); ok {
		filter = subfilter.(string)
	}
	return fmt.Sprintf("%s-%s", project, filter)
}
