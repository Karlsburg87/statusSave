package sqlstatements

//InsertStatusUpdates fetches the service uuid from the services tables or inserts new row to that table
//  and returns to insert status update with correct foreign key
var InsertStatusUpdates = `
WITH servicer AS (
	INSERT INTO statusSentry.public.services (
	uid,
	service_name,
	domain,
	status_page) VALUES (DEFAULT,$5,$6,$7) 
	ON CONFLICT (service_name) DO NOTHING
	RETURNING uid
)
INSERT INTO statusSentry.public.statusUpdates (
	service, 
	message,
	raw_message,
	message_pub_timestamp,
	status_page) SELECT servicer.uid,$1,$2,$3,$4 FROM servicer;
	`

//https://www.cockroachlabs.com/docs/stable/insert.html#on-conflict-clause
//https://www.cockroachlabs.com/docs/stable/common-table-expressions.html
