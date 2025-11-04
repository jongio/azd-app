"""
Custom Azure Identity TokenCredential that fetches tokens from the auth server.

This credential implements the Azure SDK TokenCredential interface and can be used
as a drop-in replacement for DefaultAzureCredential or any other credential.

Example:
    from auth_credential import AuthServerCredential
    from azure.mgmt.resource import SubscriptionClient
    
    credential = AuthServerCredential(
        server_url="http://auth-server:8080",
        secret="your-secret"
    )
    
    client = SubscriptionClient(credential)
    subscriptions = list(client.subscriptions.list())
"""

import os
import time
import requests
from typing import Optional
from azure.core.credentials import AccessToken, TokenCredential


class AuthServerCredential(TokenCredential):
    """
    Azure SDK compatible credential that fetches tokens from the auth server.
    
    This credential works with any Azure SDK that accepts a TokenCredential,
    making it a drop-in replacement for DefaultAzureCredential.
    
    Args:
        server_url: URL of the auth server (e.g., "http://auth-server:8080")
        secret: Shared secret for authentication
        timeout: Request timeout in seconds (default: 30)
    
    Environment Variables:
        AUTH_SERVER_URL: Auth server URL (used if server_url not provided)
        AZD_AUTH_SECRET: Shared secret (used if secret not provided)
    """
    
    def __init__(
        self,
        server_url: Optional[str] = None,
        secret: Optional[str] = None,
        timeout: int = 30
    ):
        self.server_url = server_url or os.environ.get("AUTH_SERVER_URL")
        self.secret = secret or os.environ.get("AZD_AUTH_SECRET")
        self.timeout = timeout
        self._token_cache = {}
        
        if not self.server_url:
            raise ValueError(
                "server_url must be provided or AUTH_SERVER_URL environment variable must be set"
            )
        
        if not self.secret:
            raise ValueError(
                "secret must be provided or AZD_AUTH_SECRET environment variable must be set"
            )
        
        # Remove trailing slash from server URL
        self.server_url = self.server_url.rstrip("/")
    
    def get_token(self, *scopes: str, **kwargs) -> AccessToken:
        """
        Request an access token for the specified scopes.
        
        This method is called by Azure SDK clients to get authentication tokens.
        It fetches tokens from the auth server and caches them locally.
        
        Args:
            *scopes: The requested scopes (e.g., "https://management.azure.com/.default")
            **kwargs: Additional keyword arguments (unused, for Azure SDK compatibility)
        
        Returns:
            AccessToken: Token object with token string and expiration time
        
        Raises:
            RuntimeError: If unable to fetch token from auth server
        """
        if not scopes:
            scope = "https://management.azure.com/.default"
        else:
            scope = scopes[0]
        
        # Check cache first
        cached = self._get_cached_token(scope)
        if cached:
            return cached
        
        # Fetch new token from auth server
        try:
            token_data = self._fetch_token(scope)
            
            # Parse JWT to get expiration (JWT is in format: header.payload.signature)
            # For simplicity, we'll use expires_in from response
            expires_on = int(time.time()) + token_data["expires_in"]
            
            token = AccessToken(token_data["access_token"], expires_on)
            
            # Cache the token
            self._cache_token(scope, token)
            
            return token
        
        except requests.RequestException as e:
            raise RuntimeError(f"Failed to fetch token from auth server: {e}")
        except (KeyError, ValueError) as e:
            raise RuntimeError(f"Invalid token response from auth server: {e}")
    
    def _fetch_token(self, scope: str) -> dict:
        """Fetch a token from the auth server."""
        url = f"{self.server_url}/token"
        headers = {
            "Authorization": f"Bearer {self.secret}"
        }
        params = {
            "scope": scope
        }
        
        response = requests.get(
            url,
            headers=headers,
            params=params,
            timeout=self.timeout
        )
        
        response.raise_for_status()
        return response.json()
    
    def _get_cached_token(self, scope: str) -> Optional[AccessToken]:
        """Get a token from the cache if it's still valid."""
        if scope not in self._token_cache:
            return None
        
        token = self._token_cache[scope]
        
        # Check if token is expired (with 60 second buffer)
        if token.expires_on <= int(time.time()) + 60:
            # Token expired or about to expire
            del self._token_cache[scope]
            return None
        
        return token
    
    def _cache_token(self, scope: str, token: AccessToken):
        """Cache a token for future use."""
        self._token_cache[scope] = token
    
    def close(self):
        """Clean up resources. Called by Azure SDK when credential is no longer needed."""
        self._token_cache.clear()
    
    def __enter__(self):
        """Context manager entry."""
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit."""
        self.close()
