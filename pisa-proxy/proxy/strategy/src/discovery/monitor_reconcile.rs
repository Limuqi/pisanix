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

use crossbeam_channel::unbounded;
use tracing::error;

use crate::{
    config::ReadWriteSplittingDynamic,
    readwritesplitting::{dynamic_rw::MonitorResponse, ReadWriteEndpoint},
};

pub struct MonitorReconcile {
    config: crate::config::Discovery,
    rw_endpoint: ReadWriteEndpoint,
}

use crate::monitors::{
    connect_monitor::ConnectMonitorResponse, ping_monitor::PingMonitorResponse,
    read_only_monitor::ReadOnlyMonitorResponse,
    replication_lag_monitor::ReplicationLagMonitorResponse,
};

impl MonitorReconcile {
    pub fn new(config: ReadWriteSplittingDynamic, rw_endpoint: ReadWriteEndpoint) -> Self {
        MonitorReconcile { config: config.discovery, rw_endpoint }
    }

    pub fn start_monitor_reconcile(
        &mut self,
        monitor_interval: u64,
        monitor_response_channel: crate::readwritesplitting::MonitorResponseChannel,
        monitors_len: usize,
    ) -> crossbeam_channel::Receiver<ReadWriteEndpoint> {
        let (send, recv) = unbounded();
        let tx = send.clone();
        let rx = recv.clone();

        let rw_endpoint = self.rw_endpoint.clone();

        tokio::spawn(async move {
            MonitorReconcile::report(
                tx,
                monitor_interval,
                rw_endpoint,
                monitor_response_channel,
                monitors_len,
            )
            .await;
        });

        rx
    }

    async fn report(
        s: crossbeam_channel::Sender<ReadWriteEndpoint>,
        monitor_interval: u64,
        rw_endpoint: ReadWriteEndpoint,
        monitor_response_channel: crate::readwritesplitting::MonitorResponseChannel,
        monitors_len: usize,
    ) {
        let mut connect_monitor_response: Option<ConnectMonitorResponse> = None;
        let mut ping_monitor_response: Option<PingMonitorResponse> = None;
        let mut replication_lag_monitor_response: Option<ReplicationLagMonitorResponse> = None;
        let mut read_only_monitor_response: Option<ReadOnlyMonitorResponse> = None;

        tokio::task::spawn_blocking(move || {
            let mut pre_rw_endpoint = rw_endpoint.clone();
            loop {
                let monitor_response_channel = monitor_response_channel.clone();
                let mut curr_rw_endpoint = rw_endpoint.clone();
                for _ in 0..monitors_len {
                    match monitor_response_channel.monitor_response_rx.recv().unwrap() {
                        MonitorResponse::ConnectMonitorResponse(connect_response) => {
                            connect_monitor_response = Some(connect_response);
                        }
                        MonitorResponse::PingMonitorResponse(ping_response) => {
                            ping_monitor_response = Some(ping_response);
                        }
                        MonitorResponse::ReadOnlyMonitorResponse(read_only_response) => {
                            read_only_monitor_response = Some(read_only_response);
                        }
                        MonitorResponse::ReplicationLagResponse(replication_lag_response) => {
                            replication_lag_monitor_response = Some(replication_lag_response);
                        }
                    }
                }

                for (_read_write_connect_addr, read_write_connect_status) in
                    &connect_monitor_response.as_ref().unwrap().readwrite
                {
                    match read_write_connect_status {
                        // check master connected
                        crate::monitors::connect_monitor::ConnectStatus::Connected => {
                            for (_read_write_ping_addr, read_write_ping_status) in
                                ping_monitor_response.clone().unwrap().readwrite
                            {
                                match read_write_ping_status {
                                    // if master connected is ok, check ping
                                    crate::monitors::ping_monitor::PingStatus::PingOk => {}
                                    crate::monitors::ping_monitor::PingStatus::PingNotOk => {
                                        if curr_rw_endpoint.clone().read.len() > 0 {
                                            //check if slave is change to master
                                            for read_endpoint in curr_rw_endpoint.clone().read {
                                                match read_only_monitor_response.as_ref().unwrap().roles.get(&read_endpoint.addr).unwrap() {
                                                    // slave change to master
                                                    crate::monitors::read_only_monitor::NodeRole::Master => {
                                                        // clean readwrite list
                                                        curr_rw_endpoint.readwrite = vec![];
                                                        // add new read write into master list
                                                        curr_rw_endpoint.readwrite.push(read_endpoint);
                                                    },
                                                    // slave doesn't change to master
                                                    crate::monitors::read_only_monitor::NodeRole::Slave => {
                                                        // nothing to do
                                                        continue;
                                                    },
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                        }
                        // master node connected failed
                        crate::monitors::connect_monitor::ConnectStatus::Disconnected => {
                            // check if slave is change to master
                            for read_endpoint in curr_rw_endpoint.clone().read {
                                match read_only_monitor_response.clone() {
                                    Some(read_only_response) => {
                                        match read_only_response.roles.get(&read_endpoint.addr).unwrap() {
                                            //slave change to master
                                            crate::monitors::read_only_monitor::NodeRole::Master => {
                                                curr_rw_endpoint.readwrite = vec![];
                                                // add new read write into master list
                                                curr_rw_endpoint.readwrite.push(read_endpoint);
                                            }
                                            // slave doesn't change to master
                                            crate::monitors::read_only_monitor::NodeRole::Slave => {
                                            }
                                        }
                                    }
                                    None => {}
                                }
                            }
                        }
                    }
                }
                for (read_addr, read_connect_status) in
                    &connect_monitor_response.as_ref().unwrap().read
                {
                    match read_connect_status {
                        crate::monitors::connect_monitor::ConnectStatus::Connected => {
                            match ping_monitor_response.clone() {
                                Some(ping_response) => {
                                    for (read_ping_addr, read_ping_status) in
                                        ping_response.clone().read
                                    {
                                        match read_ping_status {
                                            crate::monitors::ping_monitor::PingStatus::PingOk => {
                                                match replication_lag_monitor_response.clone() {
                                                    Some(replication_lag_response) => {
                                                        for (replication_lag_addr, lag_status) in
                                                            &replication_lag_response.latency
                                                        {
                                                            if !lag_status.is_latency {
                                                                match curr_rw_endpoint.read.iter().find(
                                                                    |r| r.addr.eq(replication_lag_addr),
                                                                ) {
                                                                    Some(_) => {}
                                                                    None => {
                                                                        curr_rw_endpoint.read.append(
                                                                            &mut curr_rw_endpoint
                                                                                .read
                                                                                .clone(),
                                                                        );
                                                                    }
                                                                }
                                                                continue;
                                                            } else {
                                                                // add replication_lag_addr to read_write list
                                                                curr_rw_endpoint.read.remove(
                                                                    rw_endpoint
                                                                        .read
                                                                        .iter()
                                                                        .position(|r| {
                                                                            r.addr.eq(replication_lag_addr)
                                                                        })
                                                                        .unwrap(),
                                                                );
                                                            }
                                                        }
                                                    }
                                                    None => {}
                                                }
                                            }
                                            crate::monitors::ping_monitor::PingStatus::PingNotOk => {
                                                curr_rw_endpoint.read.remove(
                                                    rw_endpoint
                                                        .read
                                                        .iter()
                                                        .position(|r| r.addr.eq(&read_ping_addr))
                                                        .unwrap(),
                                                );
                                            }
                                        }
                                    }
                                }
                                None => {}
                            }
                        }
                        crate::monitors::connect_monitor::ConnectStatus::Disconnected => {
                            curr_rw_endpoint.read.remove(
                                rw_endpoint.read.iter().position(|r| r.addr.eq(read_addr)).unwrap(),
                            );
                        }
                    }
                }

                if pre_rw_endpoint != curr_rw_endpoint {
                    if let Err(err) = s.send(curr_rw_endpoint.clone()) {
                        error!("send read write endpoint err: {:#?}", err);
                    }
                }

                pre_rw_endpoint = curr_rw_endpoint;

                std::thread::sleep(std::time::Duration::from_millis(monitor_interval));
            }
        });
    }
}
