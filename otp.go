package main

import (
	"context"
	"time"
	
	"github.com/google/uuid"
)

type OTP struct {
	Key 	string
	Created time.Time 
}

type RetentionMap map[string]OTP // retention map will delete any (OTP) One Time Password that are too old

func NewRetentionMap(ctx context.Context, retentionPeriod time.Duration) RetentionMap {
	rm := make(RetentionMap)
	
	// when new retentionmap created also create background process to check for expired OTP's
	go rm.Retention(ctx, retentionPeriod)
	
	return rm
}


// create new OTP
func (rm RetentionMap) NewOTP() OTP {
	o := OTP {
		Key: uuid.NewString(),
		Created: time.Now(),
	}
	
	rm[o.Key] = o 
	return o 
}

// VerifyOTP will accept one otp string check if it exists and return true if it does false if it does not
// if does exist delete otp if it has been used
func (rm RetentionMap) VerifyOTP(otp string) bool {
	if _, ok := rm[otp]; !ok {
		return false // otp is not valid
	}
	delete(rm, otp)
	return true
}

func (rm RetentionMap) Retention(ctx context.Context, retentionPeriod time.Duration) {
	ticker := time.NewTicker(400 * time.Millisecond) // how often we will be checking all of the OTP's
	
	// each time the ticker ticks, loop through all OTP's 
	for {
		select {
		case <-ticker.C:
			for _, otp := range rm {
				// take opt created time add retentionPeriod to know time when OTP is no longer valid
				// check if time is before time.now if it is then password is no longer valid and must remove
				if otp.Created.Add(retentionPeriod).Before(time.Now()) {
					delete(rm, otp.Key) // remove OTP when no longer valid
				}
			}
		case <-ctx.Done(): // return to cancel when closed
			return 
		}
	}
}