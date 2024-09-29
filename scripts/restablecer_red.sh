#!/bin/sh

#  restablecer_red.sh
#

sudo rm -f /etc/machine-id
sudo dbus-uuidgen --ensure=/etc/machine-id
sudo reboot

#
#  Created by Adrian Gil on 26/7/24.
#  
