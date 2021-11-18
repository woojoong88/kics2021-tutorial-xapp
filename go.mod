module sample-xapp

go 1.15

require (
	github.com/gogo/protobuf v1.3.2
	github.com/onosproject/onos-api/go v0.7.85
	github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm_v2 v0.0.0-00010101000000-000000000000
	github.com/onosproject/onos-lib-go v0.7.10
	github.com/onosproject/onos-ric-sdk-go v0.7.22
	golang.org/x/mod v0.3.1-0.20200828183125-ce943fd02449 // indirect
	golang.org/x/tools v0.1.0 // indirect
	google.golang.org/protobuf v1.26.0
	honnef.co/go/tools v0.0.1-2020.1.3 // indirect
)

replace github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm_v2 => /home/vagrant/onos-e2-sm/servicemodels/e2sm_kpm_v2
