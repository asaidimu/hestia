// Package model holds the parked tenant half-feature (audit A-13): the
// SystemTenant model, its collection and its migration chain exist, but no
// service, handler, or registration is wired for tenants — the feature was
// left half-landed by the original build-out.
//
// Decision (2026-08-30, audit A-13): park, do not finish piecemeal. The model
// and migrations stay so the tenant_id column and schema history remain
// stable for deployments that already carry the collection, and the runtime
// TenantDispatcher already scopes requests by the authenticated claims'
// TenantID, so the data path is tenant-ready. Actual tenant administration
// (service, handlers, policy bindings, UI) should land as one scoped piece
// of work when multi-tenancy is genuinely required — extending this package
// bit by bit before that decision is how the half-feature happened in the
// first place.
package model
