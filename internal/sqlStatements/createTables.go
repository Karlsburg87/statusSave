package sqlstatements

//CreateTables creates the initial tables for the service using Cockroach DB flavour SQL (notice STRING types)
var CreateTables = `


CREATE TABLE IF NOT EXISTS statussentry.public.services(
	uid uuid DEFAULT gen_random_uuid(),
	service_name STRING UNIQUE,
	domain STRING,
	status_page STRING,
	PRIMARY KEY(uid)
);

CREATE TABLE IF NOT EXISTS statussentry.public.statusUpdates(
	"id" INT DEFAULT unique_rowid(),
	service UUID NOT NULL REFERENCES statussentry.public.services (uid) ON UPDATE CASCADE ON DELETE CASCADE,
	message STRING,
	raw_message STRING,
	message_pub_timestamp TIMESTAMPTZ,
	status_page STRING,
	PRIMARY KEY (id),
	INDEX (service)
	);

CREATE TABLE IF NOT EXISTS statussentry.public.pingPolls(
	"id" INT DEFAULT unique_rowid(),
	service UUID NOT NULL REFERENCES statussentry.public.services (uid) ON UPDATE CASCADE ON DELETE CASCADE,  
	pinged_url STRING,
	status_code INT,
	error_text STRING,
	poll_timestamp TIMESTAMPTZ,
	dns_polltime INT,
	tls_handshake_polltime INT,
	connect_polltime INT,
	first_response_polltime INT,
	cert_verified BOOL,
	cert_valid_from DATE,
	cert_valid_until DATE,
	issuer STRING,
	PRIMARY KEY (id),
	INDEX (service)
);
`
