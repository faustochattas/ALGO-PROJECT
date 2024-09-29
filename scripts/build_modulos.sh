#!/bin/sh

#  moverse_kernel.sh

cd ..
cd kernel/
go build kernel.go

cd ..
cd memoria/
go build memoria.go

cd ..
cd entradasalida/
go build entradasalida.go

cd ..
cd cpu/
go build cpu.go


#  
#
#  Created by Adrian Gil on 25/7/24.
#  
