# Modbus Client Completion Trigger

This trigger listens for request-completion events published by the
`activity/mbclient` activity in the same Flogo runtime. It has no settings and
forwards each event to its configured handlers. Events contain the activity
name, input request, decoded value on success, and an error message on failure.

The trigger must be included in the same application as `activity/mbclient`.
