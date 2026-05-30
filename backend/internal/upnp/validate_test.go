package upnp

import "testing"

func TestValidatePortMapping(t *testing.T) {
	tests := []struct {
		name    string
		in      PortMapping
		wantErr bool // ValidatePortMapping() error presence
	}{
		{
			name: "valid tcp",
			in: PortMapping{
				Protocol:     "TCP",
				ExternalPort: 8080,
				InternalIP:   "192.168.1.20",
				InternalPort: 8080,
			},
			wantErr: false,
		},
		{
			name: "valid udp",
			in: PortMapping{
				Protocol:     "UDP",
				ExternalPort: 5353,
				InternalIP:   "192.168.1.20",
				InternalPort: 5353,
			},
			wantErr: false,
		},
		{
			name: "invalid protocol",
			in: PortMapping{
				Protocol:     "ICMP",
				ExternalPort: 8080,
				InternalIP:   "192.168.1.20",
				InternalPort: 8080,
			},
			wantErr: true,
		},
		{
			name: "external port out of range",
			in: PortMapping{
				Protocol:     "TCP",
				ExternalPort: 70000,
				InternalIP:   "192.168.1.20",
				InternalPort: 8080,
			},
			wantErr: true,
		},
		{
			name: "missing internal ip",
			in: PortMapping{
				Protocol:     "TCP",
				ExternalPort: 8080,
				InternalPort: 8080,
			},
			wantErr: true,
		},
		{
			name: "invalid internal ip",
			in: PortMapping{
				Protocol:     "TCP",
				ExternalPort: 8080,
				InternalIP:   "999.999.999.999",
				InternalPort: 8080,
			},
			wantErr: true,
		},
		{
			name: "empty description allowed",
			in: PortMapping{
				Protocol:     "TCP",
				ExternalPort: 8080,
				InternalIP:   "192.168.1.20",
				InternalPort: 8080,
				Description:  "",
			},
			wantErr: false,
		},
		{
			name: "negative lease duration",
			in: PortMapping{
				Protocol:             "TCP",
				ExternalPort:         8080,
				InternalIP:           "192.168.1.20",
				InternalPort:         8080,
				LeaseDurationSeconds: -1,
			},
			wantErr: true,
		},
		{
			name: "huge lease duration",
			in: PortMapping{
				Protocol:             "TCP",
				ExternalPort:         8080,
				InternalIP:           "192.168.1.20",
				InternalPort:         8080,
				LeaseDurationSeconds: 99999999,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidatePortMapping(tt.in); (err != nil) != tt.wantErr {
				t.Fatalf("ValidatePortMapping() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
