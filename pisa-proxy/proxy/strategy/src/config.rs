// Copyright 2022 SphereEx Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

use loadbalance::balance::AlgorithmName;
use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize, Clone, Default)]
pub struct ReadWriteSplitting {
    #[serde(rename = "static")]
    pub statics: Option<ReadWriteSplittingStatic>,
    pub dynamic: Option<ReadWriteSplittingDynamic>,
}

#[derive(Debug, Serialize, Deserialize, Clone, Default)]
pub struct ReadWriteSplittingStatic {
    pub default_target: TargetRole,
    #[serde(rename = "rule")]
    pub rules: Vec<ReadWriteSplittingRule>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(untagged)]
pub enum ReadWriteSplittingRule {
    Regex(RegexRule),
    Generic(GenericRule),
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct ReadWriteSplittingDynamic {
    pub default_target: TargetRole,
    #[serde(rename = "rule")]
    pub rules: Vec<ReadWriteSplittingRule>,
    pub discovery: Discovery,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(rename_all = "lowercase", tag = "type")]
pub enum Discovery {
    Mha(MasterHighAvailability),
}

#[derive(Debug, Serialize, Deserialize, Clone, PartialEq, Default)]
pub struct MasterHighAvailability {
    pub user: String,
    pub password: String,
    #[serde(default = "default_monitor_period")]
    pub monitor_period: u64,
    #[serde(default = "default_connect_period")]
    pub connect_period: u64,
    #[serde(default = "default_connect_timeout")]
    pub connect_timeout: u64,
    #[serde(default = "default_connect_failure_threshold")]
    pub connect_failure_threshold: u64,
    #[serde(default = "default_ping_period")]
    pub ping_period: u64,
    #[serde(default = "default_ping_timeout")]
    pub ping_timeout: u64,
    #[serde(default = "default_ping_failure_threshold")]
    pub ping_failure_threshold: u64,
    #[serde(default = "default_replication_lag_period")]
    pub replication_lag_period: u64,
    #[serde(default = "default_replication_lag_timeout")]
    pub replication_lag_timeout: u64,
    #[serde(default = "default_replication_lag_failure_threshold")]
    pub replication_lag_failure_threshold: u64,
    #[serde(default = "default_max_replication_lag")]
    pub max_replication_lag: u64,
    #[serde(default = "default_read_only_period")]
    pub read_only_period: u64,
    #[serde(default = "default_read_only_timeout")]
    pub read_only_timeout: u64,
    #[serde(default = "default_read_only_failure_threshold")]
    pub read_only_failure_threshold: u64,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct RegexRule {
    pub name: String,
    #[serde(rename = "type")]
    pub rule_type: String,
    pub regex: Vec<String>,
    pub target: TargetRole,
    pub algorithm_name: AlgorithmName,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct GenericRule {
    pub name: String,
    #[serde(rename = "type")]
    pub rule_type: String,
    pub algorithm_name: AlgorithmName,
}

#[derive(Debug, Serialize, Deserialize, Clone, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum TargetRole {
    Read,
    ReadWrite,
}

impl Default for TargetRole {
    fn default() -> Self {
        Self::ReadWrite
    }
}

fn default_monitor_period() -> u64 {
    1000
}

fn default_connect_period() -> u64 {
    1000
}

fn default_connect_timeout() -> u64 {
    6000
}

fn default_connect_failure_threshold() -> u64 {
    1
}

fn default_ping_period() -> u64 {
    1000
}

fn default_ping_timeout() -> u64 {
    6000
}

fn default_ping_failure_threshold() -> u64 {
    1
}

fn default_replication_lag_period() -> u64 {
    1000
}

fn default_replication_lag_timeout() -> u64 {
    6000
}

fn default_replication_lag_failure_threshold() -> u64 {
    1
}

fn default_max_replication_lag() -> u64 {
    10000
}

fn default_read_only_period() -> u64 {
    1000
}

fn default_read_only_timeout() -> u64 {
    6000
}

fn default_read_only_failure_threshold() -> u64 {
    1
}
