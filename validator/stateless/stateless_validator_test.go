/*
 * Copyright (C) 2018 The ZeepinChain Authors
 * This file is part of The ZeepinChain library.
 *
 * The ZeepinChain is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The ZeepinChain is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The ZeepinChain.  If not, see <http://www.gnu.org/licenses/>.
 */
package stateless

import (
	"testing"

	"github.com/ontio/ontology-crypto/keypair"
	"github.com/ontio/ontology-eventbus/actor"
	"github.com/mileschao/ZeepinChain/account"
	"github.com/mileschao/ZeepinChain/common/log"
	"github.com/mileschao/ZeepinChain/core/signature"
	ctypes "github.com/mileschao/ZeepinChain/core/types"
	"github.com/mileschao/ZeepinChain/core/utils"
	"github.com/mileschao/ZeepinChain/errors"
	types2 "github.com/mileschao/ZeepinChain/validator/types"
	"github.com/stretchr/testify/assert"
	"time"
)

func signTransaction(signer *account.Account, tx *ctypes.Transaction) error {
	hash := tx.Hash()
	sign, _ := signature.Sign(signer, hash[:])
	tx.Sigs = append(tx.Sigs, &ctypes.Sig{
		PubKeys: []keypair.PublicKey{signer.PublicKey},
		M:       1,
		SigData: [][]byte{sign},
	})
	return nil
}

func TestStatelessValidator(t *testing.T) {
	log.Init(log.PATH, log.Stdout)
	acc := account.NewAccount("")

	code := []byte{1, 2, 3}

	tx := utils.NewDeployTransaction(code, "test", "1", "author", "author@123.com", "test desp", false)

	tx.Payer = acc.Address

	signTransaction(acc, tx)

	validator := &validator{id: "test"}
	props := actor.FromProducer(func() actor.Actor {
		return validator
	})

	pid, err := actor.SpawnNamed(props, validator.id)
	assert.Nil(t, err)

	msg := &types2.CheckTx{WorkerId: 1, Tx: *tx}
	fut := pid.RequestFuture(msg, time.Second)

	res, err := fut.Result()
	assert.Nil(t, err)

	result := res.(*types2.CheckResponse)
	assert.Equal(t, result.ErrCode, errors.ErrNoError)
	assert.Equal(t, tx.Hash(), result.Hash)
}
