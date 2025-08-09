#!/usr/bin/env rust-script
//! ```cargo
//! [dependencies]
//! serde = { version = "1.0", features = ["derive"] }
//! serde_json = "1.0"
//! reqwest = { version = "0.11", features = ["blocking", "json"] }
//! uuid = { version = "1.0", features = ["v4"] }
//! rand = "0.8"
//! ```

use std::env;
use std::io::{self, Read};
use std::fs::{File, OpenOptions};
use std::io::{BufRead, BufReader, Write};
use std::path::Path;
use std::time::{SystemTime, UNIX_EPOCH};
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use uuid::Uuid;
use rand::Rng;

#[derive(Debug, Deserialize)]
struct ToolData {
    tool_name: String,
    tool_input: Option<Value>,
    tool_output: Option<Value>,
    event: String,
    timestamp: String,
    tool_call_id: Option<String>,
}

#[derive(Debug, Serialize, Clone)]
struct AgUIEvent {
    #[serde(rename = "type")]
    event_type: String,
    data: EventData,
}

#[derive(Debug, Serialize, Clone)]
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
    // Check circuit breaker first
    if is_circuit_breaker_open() {
        eprintln!("Circuit breaker is open, skipping hook execution");
        return Ok(());
    }

    // Read tool data from stdin
    let mut input = String::new();
    io::stdin().read_to_string(&mut input)?;
    
    // Parse the JSON data
    let tool_data: ToolData = match serde_json::from_str(&input) {
        Ok(data) => data,
        Err(e) => {
            eprintln!("Failed to parse tool data: {}", e);
            record_failure();
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
    let batch_size: usize = env::var("ALPINE_TOOL_CALL_BATCH_SIZE")
        .unwrap_or_else(|_| "10".to_string())
        .parse()
        .unwrap_or(10);
    let sample_rate: u32 = env::var("ALPINE_TOOL_CALL_SAMPLE_RATE")
        .unwrap_or_else(|_| "100".to_string())
        .parse()
        .unwrap_or(100);
    
    // Apply sampling - skip event if random number is above sample rate
    if sample_rate < 100 {
        let mut rng = rand::thread_rng();
        let random_value: u32 = rng.gen_range(1..=100);
        if random_value > sample_rate {
            eprintln!("Event sampled out ({}% rate)", sample_rate);
            return Ok(());
        }
    }
    
    // Generate or use existing tool call ID
    let tool_call_id = tool_data.tool_call_id
        .unwrap_or_else(|| Uuid::new_v4().to_string());
    
    // Determine event type based on whether we have tool output
    let event_type = if tool_data.tool_output.is_some() {
        "ToolCallEnd"
    } else {
        "ToolCallStart"
    };
    
    // Create event
    let event = AgUIEvent {
        event_type: event_type.to_string(),
        data: EventData {
            tool_call_id: tool_call_id.clone(),
            tool_call_name: tool_data.tool_name.clone(),
            run_id: run_id.clone(),
            tool_input: if event_type == "ToolCallStart" { tool_data.tool_input } else { None },
            tool_output: if event_type == "ToolCallEnd" { tool_data.tool_output } else { None },
        },
    };
    
    // Handle batching with error handling
    let result = if batch_size > 1 {
        add_to_batch(&event, batch_size, &endpoint)
            .or_else(|e| {
                eprintln!("Failed to add event to batch: {}, trying direct send", e);
                send_event(&endpoint, &event)
            })
    } else {
        send_event(&endpoint, &event)
    };

    match result {
        Ok(_) => {
            record_success();
            eprintln!("Event sent successfully");
        }
        Err(e) => {
            eprintln!("Failed to send event: {}", e);
            record_failure();
            // Don't fail the hook - workflow should continue
        }
    }
    
    Ok(())
}

fn add_to_batch(event: &AgUIEvent, batch_size: usize, endpoint: &str) -> Result<(), Box<dyn std::error::Error>> {
    let batch_file = "/tmp/alpine_event_batch.json";
    
    // Read existing batch or create new one
    let mut events: Vec<AgUIEvent> = if Path::new(batch_file).exists() {
        let file = File::open(batch_file)?;
        let reader = BufReader::new(file);
        let mut batch_events = Vec::new();
        
        for line in reader.lines() {
            if let Ok(line) = line {
                if let Ok(event) = serde_json::from_str::<AgUIEvent>(&line) {
                    batch_events.push(event);
                }
            }
        }
        batch_events
    } else {
        Vec::new()
    };
    
    // Add current event to batch
    events.push(event.clone());
    
    // Check if batch is full
    if events.len() >= batch_size {
        // Send batch
        if let Err(e) = send_batch(endpoint, &events) {
            eprintln!("Failed to send batch: {}", e);
        }
        
        // Clear batch file
        std::fs::remove_file(batch_file).ok();
    } else {
        // Write updated batch back to file
        let mut file = OpenOptions::new()
            .create(true)
            .write(true)
            .truncate(true)
            .open(batch_file)?;
        
        for event in &events {
            let event_json = serde_json::to_string(event)?;
            writeln!(file, "{}", event_json)?;
        }
    }
    
    Ok(())
}

fn send_batch(endpoint: &str, events: &[AgUIEvent]) -> Result<(), Box<dyn std::error::Error>> {
    let client = reqwest::blocking::Client::builder()
        .timeout(std::time::Duration::from_secs(10))
        .build()?;
    
    let batch_payload = json!({
        "events": events
    });
    
    let response = client
        .post(endpoint)
        .json(&batch_payload)
        .send()?;
    
    if !response.status().is_success() {
        return Err(format!("HTTP error: {}", response.status()).into());
    }
    
    eprintln!("Sent batch of {} events", events.len());
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