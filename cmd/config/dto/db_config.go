package dto

type DataBaseConfigInterface interface {
	Host() string
	Port() int
	Name() string
	User() string
	Password() string
	Timezone() string
	SetHost(host string)
	SetPort(port int)
	SetName(name string)
	SetUser(user string)
	SetPassword(password string)
	SetTimezone(timezone string)
}

type DatabaseConfig struct {
	host     string
	port     int
	name     string
	user     string
	password string
	timezone string
}

func (d *DatabaseConfig) Host() string {
	return d.host
}

func (d *DatabaseConfig) Port() int {
	return d.port
}

func (d *DatabaseConfig) Name() string {
	return d.name
}

func (d *DatabaseConfig) User() string {
	return d.user
}

func (d *DatabaseConfig) Password() string {
	return d.password
}

func (d *DatabaseConfig) Timezone() string {
	return d.timezone
}

func (d *DatabaseConfig) SetHost(host string) {
	d.host = host
}

func (d *DatabaseConfig) SetPort(port int) {
	d.port = port
}

func (d *DatabaseConfig) SetName(name string) {
	d.name = name
}

func (d *DatabaseConfig) SetUser(user string) {
	d.user = user
}

func (d *DatabaseConfig) SetPassword(password string) {
	d.password = password
}

func (d *DatabaseConfig) SetTimezone(timezone string) {
	d.timezone = timezone
}
