#!/usr/bin/env rust-script
//! ```cargo
//! [dependencies]
//! serde = { version = "1.0", features = ["derive"] }
//! serde_json = "1.0"
//! reqwest = { version = "0.11", features = ["blocking", "json"] }
//! uuid = { version = "1.0", features = ["v4"] }
//! ```

use std::env;
use std::io::{self, Read};
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use uuid::Uuid;

#[derive(Debug, Deserialize)]
struct ToolData {
    tool_name: String,
    tool_input: Option<Value>,
    tool_output: Option<Value>,
    event: String,
    timestamp: String,
    tool_call_id: Option<String>,
}

#[derive(Debug, Serialize)]
struct AgUIEvent {
    #[serde(rename = "type")]
    event_type: String,
    data: EventData,
}

#[derive(Debug, Serialize)]
struct EventData {
    #[serde(rename = "toolCallId")]
    tool_call_id: String,
    #[serde(rename = "toolCallName")]
    tool_call_name: String,
    #[serde(rename = "runId")]
    run_id: String,
    #[serde(rename = "toolInput", skip_serializing_if = "Option::is_none")]
    tool_input: Option<Value>,
    #[serde(rename = "toolOutput", skip_serializing_if = "Option::is_none")]
    tool_output: Option<Value>,
}

fn main() -> io::Result<()> {
    // Read tool data from stdin
    let mut input = String::new();
    io::stdin().read_to_string(&mut input)?;
    
    // Parse the JSON data
    let tool_data: ToolData = match serde_json::from_str(&input) {
        Ok(data) => data,
        Err(e) => {
            eprintln!("Failed to parse tool data: {}", e);
            return Ok(()); // Don't fail the hook
        }
    };
    
    // Log that hook was called
    eprintln!("HOOK CALLED: tool={}", tool_data.tool_name);
    
    // Get environment variables
    let endpoint = match env::var("ALPINE_EVENTS_ENDPOINT") {
        Ok(val) => val,
        Err(_) => {
            eprintln!("ALPINE_EVENTS_ENDPOINT not set, skipping event emission");
            return Ok(());
        }
    };
    
    let run_id = env::var("ALPINE_RUN_ID").unwrap_or_else(|_| "unknown".to_string());
    
    // Generate or use existing tool call ID
    let tool_call_id = tool_data.tool_call_id
        .unwrap_or_else(|| Uuid::new_v4().to_string());
    
    // Create ToolCallStart event
    let start_event = AgUIEvent {
        event_type: "ToolCallStart".to_string(),
        data: EventData {
            tool_call_id: tool_call_id.clone(),
            tool_call_name: tool_data.tool_name.clone(),
            run_id: run_id.clone(),
            tool_input: tool_data.tool_input.clone(),
            tool_output: None,
        },
    };
    
    // Send ToolCallStart event
    if let Err(e) = send_event(&endpoint, &start_event) {
        eprintln!("Failed to send ToolCallStart event: {}", e);
        // Continue execution even if sending fails
    }
    
    // If we have tool output, also send ToolCallEnd event
    if tool_data.tool_output.is_some() {
        let end_event = AgUIEvent {
            event_type: "ToolCallEnd".to_string(),
            data: EventData {
                tool_call_id: tool_call_id.clone(),
                tool_call_name: tool_data.tool_name.clone(),
                run_id: run_id.clone(),
                tool_input: None,
                tool_output: tool_data.tool_output,
            },
        };
        
        if let Err(e) = send_event(&endpoint, &end_event) {
            eprintln!("Failed to send ToolCallEnd event: {}", e);
        }
    }
    
    Ok(())
}

fn send_event(endpoint: &str, event: &AgUIEvent) -> Result<(), Box<dyn std::error::Error>> {
    let client = reqwest::blocking::Client::builder()
        .timeout(std::time::Duration::from_secs(5))
        .build()?;
    
    let response = client
        .post(endpoint)
        .json(event)
        .send()?;
    
    if !response.status().is_success() {
        return Err(format!("HTTP error: {}", response.status()).into());
    }
    
    Ok(())
}