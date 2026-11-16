package message

import (
	"chickchirick-messages/internal/controller/c_controller"
	"chickchirick-messages/internal/middleware"
	msgmodel "chickchirick-messages/internal/model/message"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type E2EEController struct {
	Controller c_controller.Controller
}

func (ec *E2EEController) RegisterRoutes() {
	e := ec.Controller.E
	auth := middleware.Auth(ec.Controller.DI.AuthClient)

	// Бандл / identity
	e.PUT("/e2ee/public-key", auth, ec.PutBundle)
	e.GET("/e2ee/public-key/:id", auth, ec.GetBundle)
	e.GET("/e2ee/prekeys/count", auth, ec.CountMyPreKeys)
	e.POST("/e2ee/prekeys", auth, ec.UploadPreKeys)
}

type putBundleRequest struct {
	// Совместимость со старым фронтом
	PublicKey string `json:"publicKey"`

	IdentityKey     string `json:"identityKey"`
	SignedPreKey    string `json:"signedPreKey"`
	SignedPreKeyID  int    `json:"signedPreKeyId"`
	SignedPreKeySig string `json:"signedPreKeySig"`
	OneTimePreKeys  []struct {
		KeyID     int    `json:"keyId"`
		PublicKey string `json:"publicKey"`
	} `json:"oneTimePreKeys"`
}

type bundleResponse struct {
	UserID          int    `json:"userId"`
	IdentityKey     string `json:"identityKey"`
	PublicKey       string `json:"publicKey"` // alias для старого клиента
	SignedPreKey    string `json:"signedPreKey,omitempty"`
	SignedPreKeyID  int    `json:"signedPreKeyId,omitempty"`
	SignedPreKeySig string `json:"signedPreKeySig,omitempty"`
	OneTimePreKey   *struct {
		KeyID     int    `json:"keyId"`
		PublicKey string `json:"publicKey"`
	} `json:"oneTimePreKey,omitempty"`
}

func (ec *E2EEController) resolveLocalUserID(c *gin.Context) (int, error) {
	uuidVal, ok := c.Get("user_uuid")
	if !ok {
		return 0, errors.New("no user_uuid in context")
	}
	uuidStr, _ := uuidVal.(string)
	if uuidStr == "" {
		return 0, errors.New("empty user_uuid")
	}
	ur, err := msgmodel.GetUserRelationByUuid(c.Request.Context(), ec.Controller.DI.DBDecorator.GDB(), uuidStr)
	if err != nil {
		return 0, err
	}
	return ur.UserId, nil
}

// PutBundle — загрузка identity + signed prekey + опционально OPK.
func (ec *E2EEController) PutBundle(c *gin.Context) {
	userID, err := ec.resolveLocalUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user relation not found: " + err.Error()})
		return
	}

	var req putBundleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body: " + err.Error()})
		return
	}

	identityKey := req.IdentityKey
	if identityKey == "" {
		identityKey = req.PublicKey
	}
	if identityKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "identityKey/publicKey required"})
		return
	}

	spk := req.SignedPreKey
	if spk == "" {
		spk = identityKey // fallback: старый клиент без SPK
	}
	spkID := req.SignedPreKeyID
	if spkID == 0 {
		spkID = 1
	}
	spkSig := req.SignedPreKeySig
	if spkSig == "" {
		spkSig = "legacy"
	}

	id := &msgmodel.E2EEIdentity{
		UserID:           userID,
		IdentityKeySPKI:  identityKey,
		SignedPreKeySPKI: spk,
		SignedPreKeyID:   spkID,
		SignedPreKeySig:  spkSig,
	}
	if err := msgmodel.UpsertE2EEIdentity(c.Request.Context(), ec.Controller.DI.DBDecorator.GDB(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(req.OneTimePreKeys) > 0 {
		keys := make([]msgmodel.E2EEOneTimePreKey, 0, len(req.OneTimePreKeys))
		now := time.Now().UTC()
		for _, k := range req.OneTimePreKeys {
			if k.PublicKey == "" {
				continue
			}
			keys = append(keys, msgmodel.E2EEOneTimePreKey{
				UserID:     userID,
				KeyID:      k.KeyID,
				PublicSPKI: k.PublicKey,
				CreatedAt:  now,
			})
		}
		if err := msgmodel.ReplaceOneTimePreKeys(c.Request.Context(), ec.Controller.DI.DBDecorator.GDB(), userID, keys); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "prekeys: " + err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "userId": userID})
}

// GetBundle — публичный бандл собеседника (+ один OPK, если есть).
func (ec *E2EEController) GetBundle(c *gin.Context) {
	idStr := c.Param("id")
	peerID, err := strconv.Atoi(idStr)
	if err != nil || peerID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	ident, err := msgmodel.GetE2EEIdentity(c.Request.Context(), ec.Controller.DI.DBDecorator.GDB(), peerID)
	if err != nil {
		if errors.Is(err, msgmodel.E2EEIdentityNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "no e2ee identity"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := bundleResponse{
		UserID:          ident.UserID,
		IdentityKey:     ident.IdentityKeySPKI,
		PublicKey:       ident.IdentityKeySPKI,
		SignedPreKey:    ident.SignedPreKeySPKI,
		SignedPreKeyID:  ident.SignedPreKeyID,
		SignedPreKeySig: ident.SignedPreKeySig,
	}

	opk, err := msgmodel.ConsumeOneTimePreKey(c.Request.Context(), ec.Controller.DI.DBDecorator.GDB(), peerID)
	if err == nil && opk != nil {
		resp.OneTimePreKey = &struct {
			KeyID     int    `json:"keyId"`
			PublicKey string `json:"publicKey"`
		}{KeyID: opk.KeyID, PublicKey: opk.PublicSPKI}
	}

	c.JSON(http.StatusOK, resp)
}

func (ec *E2EEController) CountMyPreKeys(c *gin.Context) {
	userID, err := ec.resolveLocalUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	n, err := msgmodel.CountOneTimePreKeys(c.Request.Context(), ec.Controller.DI.DBDecorator.GDB(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": n})
}

type uploadPreKeysRequest struct {
	OneTimePreKeys []struct {
		KeyID     int    `json:"keyId"`
		PublicKey string `json:"publicKey"`
	} `json:"oneTimePreKeys"`
}

func (ec *E2EEController) UploadPreKeys(c *gin.Context) {
	userID, err := ec.resolveLocalUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	var req uploadPreKeysRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	now := time.Now().UTC()
	keys := make([]msgmodel.E2EEOneTimePreKey, 0, len(req.OneTimePreKeys))
	for _, k := range req.OneTimePreKeys {
		if k.PublicKey == "" {
			continue
		}
		keys = append(keys, msgmodel.E2EEOneTimePreKey{
			UserID: userID, KeyID: k.KeyID, PublicSPKI: k.PublicKey, CreatedAt: now,
		})
	}
	// Дописываем, не затирая существующие
	if len(keys) > 0 {
		if err := ec.Controller.DI.DBDecorator.GDB().WithContext(c.Request.Context()).Create(&keys).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"uploaded": len(keys)})
}
