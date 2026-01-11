package storage

// This table is the workflow engine.
// video_jobs (
//   id
//   video_id
//   job_type           -- "validate" | "metadata" | "encode_720p"
//   state              -- pending | running | succeeded | failed | dead
//   attempt
//   lease_owner
//   lease_expires_at
//   input_uri
//   output_uri
//   created_at
//   updated_at
// )
