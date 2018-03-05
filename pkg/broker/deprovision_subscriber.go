//
// Copyright (c) 2018 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package broker

import (
	"github.com/automationbroker/bundle-lib/apb"
)

// DeprovisionWorkSubscriber - Lissten for provision messages
type DeprovisionWorkSubscriber struct {
	dao       SubscriberDAO
	msgBuffer <-chan JobMsg
}

// NewDeprovisionWorkSubscriber - Create a new work subscriber.
func NewDeprovisionWorkSubscriber(dao SubscriberDAO) *DeprovisionWorkSubscriber {
	return &DeprovisionWorkSubscriber{dao: dao}
}

// Subscribe - will start the work subscriber listenning on the message buffer for deprovision messages.
func (d *DeprovisionWorkSubscriber) Subscribe(msgBuffer <-chan JobMsg) {
	d.msgBuffer = msgBuffer
	go func() {
		log.Info("Listening for deprovision messages")
		for msg := range msgBuffer {
			log.Debug("received deprovision message from buffer")

			if _, err := d.dao.SetState(msg.InstanceUUID, msg.State); err != nil {
				log.Errorf("failed to set state after deprovision %v", err)
				continue
			}
			//only want to do this on success
			if msg.State.State == apb.StateSucceeded {
				if err := cleanupDeprovision(msg.InstanceUUID, d.dao); err != nil {
					log.Errorf("Failed cleaning up deprovision after job, error: %v", err)
					// Cleanup is reporting something has gone wrong. Deprovision overall
					// has not completed. Mark the job as failed.
					setFailedDeprovisionJob(d.dao, msg)
					continue
				}
			}
		}
	}()
}

func setFailedDeprovisionJob(dao SubscriberDAO, dmsg JobMsg) {
	// have to set the state here manually as the logic that triggers this is in the subscriber
	dmsg.State.State = apb.StateFailed
	if _, err := dao.SetState(dmsg.InstanceUUID, dmsg.State); err != nil {
		log.Errorf("failed to set state after deprovision %v", err)
	}
}

func cleanupDeprovision(id string, dao SubscriberDAO) error {
	if err := dao.DeleteServiceInstance(id); err != nil {
		log.Error("failed to delete service instance - %v", err)
		return err
	}

	return nil
}
