"""
FastAPI backend that demonstrates using the auth server with Azure SDK.

This API uses the AuthServerCredential to authenticate with Azure services
without needing direct access to Azure credentials.
"""

import os
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from azure.mgmt.resource import SubscriptionClient
from azure.mgmt.storage import StorageManagementClient
from azure.core.exceptions import ClientAuthenticationError, HttpResponseError
from auth_credential import AuthServerCredential

app = FastAPI(title="Auth Example API")

# Enable CORS for frontend
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # In production, specify exact origins
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Initialize credential
try:
    credential = AuthServerCredential()
    print("✓ Auth server credential initialized")
    print(f"  Server URL: {credential.server_url}")
except ValueError as e:
    print(f"✗ Failed to initialize credential: {e}")
    credential = None


@app.get("/")
async def root():
    """Health check endpoint."""
    return {
        "status": "healthy",
        "service": "auth-example-api",
        "auth_server": credential.server_url if credential else None
    }


@app.get("/api/subscriptions")
async def list_subscriptions():
    """
    List Azure subscriptions using the auth server credential.
    
    This demonstrates how the custom credential works with Azure SDK clients.
    """
    if not credential:
        raise HTTPException(
            status_code=500,
            detail="Credential not initialized. Check AUTH_SERVER_URL and AZD_AUTH_SECRET."
        )
    
    try:
        # Create subscription client with our custom credential
        client = SubscriptionClient(credential)
        
        # List subscriptions
        subscriptions = []
        for sub in client.subscriptions.list():
            subscriptions.append({
                "id": sub.subscription_id,
                "name": sub.display_name,
                "state": sub.state,
                "tenantId": sub.tenant_id
            })
        
        return {
            "count": len(subscriptions),
            "subscriptions": subscriptions,
            "authMethod": "AuthServerCredential"
        }
    
    except ClientAuthenticationError as e:
        raise HTTPException(
            status_code=401,
            detail=f"Authentication failed: {str(e)}"
        )
    except HttpResponseError as e:
        raise HTTPException(
            status_code=e.status_code,
            detail=f"Azure API error: {str(e)}"
        )
    except Exception as e:
        raise HTTPException(
            status_code=500,
            detail=f"Unexpected error: {str(e)}"
        )


@app.get("/api/storage-accounts")
async def list_storage_accounts():
    """
    List storage accounts in the first available subscription.
    
    This demonstrates using the credential with different Azure services.
    """
    if not credential:
        raise HTTPException(
            status_code=500,
            detail="Credential not initialized"
        )
    
    try:
        # Get first subscription
        sub_client = SubscriptionClient(credential)
        subscription = next(sub_client.subscriptions.list())
        subscription_id = subscription.subscription_id
        
        # Create storage client
        storage_client = StorageManagementClient(credential, subscription_id)
        
        # List storage accounts
        accounts = []
        for account in storage_client.storage_accounts.list():
            accounts.append({
                "name": account.name,
                "location": account.location,
                "kind": account.kind,
                "sku": account.sku.name
            })
        
        return {
            "subscription_id": subscription_id,
            "count": len(accounts),
            "accounts": accounts
        }
    
    except Exception as e:
        raise HTTPException(
            status_code=500,
            detail=f"Error listing storage accounts: {str(e)}"
        )


@app.get("/api/auth/test")
async def test_auth():
    """
    Test authentication with the auth server.
    
    This endpoint verifies that we can successfully get a token.
    """
    if not credential:
        return {
            "success": False,
            "error": "Credential not initialized"
        }
    
    try:
        # Try to get a token
        token = credential.get_token("https://management.azure.com/.default")
        
        # Don't return the actual token, just confirm we got one
        return {
            "success": True,
            "message": "Successfully authenticated with auth server",
            "expires_on": token.expires_on,
            "token_length": len(token.token)
        }
    
    except Exception as e:
        return {
            "success": False,
            "error": str(e)
        }


if __name__ == "__main__":
    import uvicorn
    
    port = int(os.environ.get("PORT", 8000))
    uvicorn.run(app, host="0.0.0.0", port=port)
