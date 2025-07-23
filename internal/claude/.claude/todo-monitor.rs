#!/usr/bin/env rust-script
//! ```cargo
//! [dependencies]
//! serde_json = "1.0"
//! ```

use serde_json::Value;
use std::env;
use std::io::{self, Read, Write};
use std::fs::File;

fn main() -> io::Result<()> {
    // Read JSON input from Claude Code
    let mut input = String::new();
    io::stdin().read_to_string(&mut input)?;
    
    let data: Value = match serde_json::from_str(&input) {
        Ok(v) => v,
        Err(_) => return Ok(()), // Exit gracefully on invalid JSON
    };
    
    // Only process TodoWrite tool
    let tool = data["tool"].as_str().unwrap_or("");
    if tool != "TodoWrite" {
        return Ok(());
    }
    
    // Extract current in_progress task
    if let Some(todos) = data["args"]["todos"].as_array() {
        for todo in todos {
            if todo["status"].as_str() == Some("in_progress") {
                if let Some(content) = todo["content"].as_str() {
                    // Write to todo file if environment variable is set
                    if let Ok(todo_file) = env::var("RIVER_TODO_FILE") {
                        if let Ok(mut file) = File::create(&todo_file) {
                            let _ = file.write_all(content.as_bytes());
                        }
                    }
                    break; // Only take the first in_progress task
                }
            }
        }
    }
    
    Ok(())
}