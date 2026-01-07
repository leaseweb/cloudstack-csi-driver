package syncer

import (
	"testing"
)

func TestCreateStorageClassName(t *testing.T) {
	cases := []struct {
		OrigName     string
		ExpectedName string
		ShouldErr    bool
	}{
		{OrigName: "cloudstack-gold", ExpectedName: "cloudstack-gold"},
		{OrigName: "cloudstack-Silver", ExpectedName: "cloudstack-silver"},
		{OrigName: "cloudstack-copper-1.2", ExpectedName: "cloudstack-copper-1.2"},
		{OrigName: "cloudstack-Custom Storage 1.2 - experimental", ExpectedName: "cloudstack-custom-storage-1.2-experimental"},
		{OrigName: "étendu", ExpectedName: "etendu"},
		{OrigName: "stockage NFS", ExpectedName: "stockage-nfs"},
		{OrigName: "Disque 123", ExpectedName: "disque-123"},
		{OrigName: "123", ExpectedName: "123"},
		{OrigName: "Platinium +", ExpectedName: "platinium"},
		{OrigName: "  Platinium Plus  ", ExpectedName: "platinium-plus"},
		{OrigName: "cloudstack-Ruthénium", ExpectedName: "cloudstack-ruthenium"},
		{OrigName: "--- gold ---", ExpectedName: "gold"},
		{OrigName: ".silver.", ExpectedName: "silver"},
		{OrigName: "Don't use me!", ExpectedName: "don-t-use-me"},
		{OrigName: "cloudstack", ExpectedName: "cloudstack"},
		{OrigName: "\u3053\u3093\u306b\u3061\u306f\u4e16\u754c", ShouldErr: true},
		{OrigName: "", ShouldErr: true},
	}

	for _, c := range cases {
		t.Run(c.OrigName, func(t *testing.T) {
			name, err := createStorageClassName(c.OrigName)
			if err != nil && !c.ShouldErr { //nolint:gocritic
				t.Error(err)
			} else if err == nil && c.ShouldErr {
				t.Error("Expected a non-nil error; error was nil")
			} else if err == nil && name != c.ExpectedName {
				t.Errorf("Expected name %s; got %s", c.ExpectedName, name)
			}
		})
	}
}
