package utils

import (
	"cloudservices/cloudmgmt/config"
	"cloudservices/common/base"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/golang/glog"
)

type recordAction string

const (
	actionUpsert = "UPSERT"
	actionDelete = "DELETE"
)

func changeRecordSet(ctx context.Context,
	action recordAction,
	svc *route53.Route53,
	zoneID string,
	dnsName string,
	ipaddr string) error {
	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String(string(action)),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(dnsName),
						ResourceRecords: []*route53.ResourceRecord{
							{
								Value: aws.String(ipaddr),
							},
						},
						TTL:  aws.Int64(60),
						Type: aws.String("A"),
					},
				},
			},
			Comment: aws.String("XI IoT edge"),
		},
		HostedZoneId: &zoneID,
	}

	result, err := svc.ChangeResourceRecordSets(input)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to %v route53 entry for dns %v, ip %v, err: %v"),
			action, dnsName, ipaddr, err)
		return err
	}

	if action == actionDelete {
		glog.Infof("dns %v, ip %v, deleted", dnsName, ipaddr)
		return nil
	}

	glog.Infof(base.PrefixRequestID(ctx, "Pending route53 %v for dns %v, ip %v"),
		action, dnsName, ipaddr)
	go func() {
		changeInput := route53.GetChangeInput{Id: result.ChangeInfo.Id}
		for true {
			changeOut, err := svc.GetChange(&changeInput)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Failed to get route53 change status entry for dns %v, ip %v, err: %v"),
					dnsName, ipaddr, err)
				return
			}
			if *changeOut.ChangeInfo.Status == route53.ChangeStatusPending {
				glog.Infof(base.PrefixRequestID(ctx, "Pending route53 change for dns %v, ip %v"),
					dnsName, ipaddr)
				time.Sleep(60 * time.Second)
			} else {
				glog.Infof(base.PrefixRequestID(ctx, "Done route53 change for dns %v, ip %v, status: %v"),
					dnsName, ipaddr, *changeOut.ChangeInfo.Status)
				return
			}
		}
	}()
	return nil
}

func findHostedZone(ctx context.Context,
	svc *route53.Route53,
	domain string) (string, error) {
	maxItems := "1"
	input := route53.ListHostedZonesByNameInput{
		DNSName:  &domain,
		MaxItems: &maxItems,
	}
	zones, err := svc.ListHostedZonesByName(&input)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get hosted zones for domain %v, err: %v"), domain, err)
		return "", err
	}

	for _, z := range zones.HostedZones {
		if *z.Name == domain {
			return *z.Id, nil
		}
	}
	return "", fmt.Errorf("Failed to find hosted zone for domain %v", domain)
}

// UpsertRoute53Entry - utility function to register a route53 A record
func UpsertRoute53Entry(ctx context.Context,
	domain string,
	dnsName string,
	ipaddr string) error {
	svc := route53.New(session.New(&aws.Config{
		Region:      aws.String("us-west-2"),
		Credentials: credentials.NewStaticCredentials(*config.Cfg.U2Route53AccessKey, *config.Cfg.U2Route53SecretKey, ""),
	}))
	// add trailing "." to domain
	zoneID, err := findHostedZone(ctx, svc, fmt.Sprintf("%v.", domain))
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get hosted zones for domain %v, err: %v"), domain, err)
		return err
	}
	return changeRecordSet(ctx, actionUpsert, svc, zoneID, dnsName, ipaddr)
}

// DeleteRoute53Entry - utility function to delete a route53 A record
func DeleteRoute53Entry(ctx context.Context,
	domain string,
	dnsName string,
	ipaddr string) error {

	glog.Infof("Deleteing route53 entry for %v %v", dnsName, ipaddr)
	svc := route53.New(session.New(&aws.Config{
		Region:      aws.String("us-west-2"),
		Credentials: credentials.NewStaticCredentials(*config.Cfg.U2Route53AccessKey, *config.Cfg.U2Route53SecretKey, ""),
	}))
	// add trailing "." to domain
	zoneID, err := findHostedZone(ctx, svc, fmt.Sprintf("%v.", domain))
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get hosted zones for domain %v, err: %v"), domain, err)
		return err
	}
	return changeRecordSet(ctx, actionDelete, svc, zoneID, dnsName, ipaddr)
}

// change alias record having the same alias target
// as the baseName record
// @param baseName example: wst-test.ntnxsherlock.com
// @param name example: wst-shyan-435e4403470385ef273eab481eaf43dc.ntnxsherlock.com
// @param action UPSERT or DELETE
func changeAliasRecord(ctx context.Context, baseName, name, action string) error {
	baseName += "."
	name += "."
	var svc *route53.Route53
	awsSession := session.New()
	if *config.Cfg.Route53CrossAccountRole == "" {
		// dev account
		svc = route53.New(awsSession)
	} else {
		// non dev account, use assume role for cross account route53 access
		config := aws.Config{
			Region:      config.Cfg.AWSRegion,
			Credentials: stscreds.NewCredentials(awsSession, *config.Cfg.Route53CrossAccountRole),
		}
		svc = route53.New(awsSession, &config)
	}

	hzOut, err := svc.ListHostedZones(nil)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to list hosted zones, err: %s"), err)
		return err
	}
	var hzId *string
	for _, hz := range hzOut.HostedZones {
		i := strings.Index(baseName, *hz.Name)
		if i != -1 {
			hzId = hz.Id
			break
		}
	}
	if hzId == nil {
		err = fmt.Errorf("Failed to find hosted zone for %s", baseName)
		glog.Errorf(base.PrefixRequestID(ctx, "Error: %s"), err)
		return err
	}
	params := &route53.ListResourceRecordSetsInput{
		HostedZoneId:    hzId,
		StartRecordName: aws.String(baseName),
		MaxItems:        aws.String("1"),
	}
	recs, err := svc.ListResourceRecordSets(params)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to list record sets: %s"), err)
		return err
	}
	if len(recs.ResourceRecordSets) != 1 {
		err = fmt.Errorf("Failed to find record for %s", baseName)
		glog.Errorf(base.PrefixRequestID(ctx, "Error: %s"), err)
		return err
	}
	rec := recs.ResourceRecordSets[0]
	rs := &route53.ResourceRecordSet{
		Name:        aws.String(name),
		Type:        rec.Type,
		AliasTarget: rec.AliasTarget,
	}
	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action:            aws.String(action),
					ResourceRecordSet: rs,
				},
			},
		},
		HostedZoneId: hzId,
	}
	result, err := svc.ChangeResourceRecordSets(input)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to change record sets: %s"), err)
		return err
	}
	glog.V(5).Infof(base.PrefixRequestID(ctx, "%s record result: %+v\n"), action, *result)
	return nil
}

// UpsertAliasRecord upsert alias record for name, using the same alias target as baseName
func UpsertAliasRecord(ctx context.Context, baseName, name string) error {
	return changeAliasRecord(ctx, baseName, name, actionUpsert)
}

// DeleteAliasRecord delete alias record for name, using the same alias target as baseName
func DeleteAliasRecord(ctx context.Context, baseName, name string) error {
	return changeAliasRecord(ctx, baseName, name, actionDelete)
}
