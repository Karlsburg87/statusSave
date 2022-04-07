package sqlstatements

//InsertPingPolls fetches the service uuid from the services tables or inserts new row to that table
// and returns to insert ping poll with correct foreign key
var InsertPingPolls = `
WITH servicer AS (
	INSERT INTO statussentry.public.services (
	uid,
	service_name,
	domain,
	status_page) VALUES (DEFAULT,$14,$15,$16) 
	ON CONFLICT (service_name,domain) DO NOTHING
	RETURNING uid
)
INSERT INTO statusSentry.public.pingPolls (
	service,
	status_page,
	pinged_url,
	status_code,
	error_text,
	poll_timestamp,
	dns_polltime,
	tls_handshake_polltime,
	connect_polltime,
	first_response_polltime,
	cert_valid_from,
	cert_valid_until,
	issuer,
	subject) SELECT servicer.uid,$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13 FROM servicer;
	`

//https://www.cockroachlabs.com/docs/stable/insert.html#on-conflict-clause
//https://www.cockroachlabs.com/docs/stable/common-table-expressions.html
