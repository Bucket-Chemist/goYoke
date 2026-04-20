use std::io;
use crate::util;
use serde::Serialize;

pub struct Config {
    pub host: String,
    pub port: u16,
}

pub enum Status {
    Active,
    Inactive,
}

pub trait Handler {
    fn handle(&self, input: &str) -> String;
}

impl Config {
    pub fn new(host: String, port: u16) -> Self {
        Config { host, port }
    }

    pub fn address(&self) -> String {
        format!("{}:{}", self.host, self.port)
    }
}

pub fn hello() -> String {
    String::from("hello")
}

fn private_helper() -> bool {
    true
}
