# Keycloak Admin Configuration for Product-Admin Service

Based on the codebase analysis, here's the exact sequence of steps you need to take in the Keycloak admin console to set up authentication for the product-admin service. Your service expects a client named "product-admin" with a "product-admin" realm role, but these are not currently defined in your realm configuration.

## Prerequisites
- Access to the Keycloak admin console (typically at `http://keycloak.local` or `http://localhost:8080` depending on your deployment)
- The "pocstore-realm" must already exist (it's imported from your realm JSON file)

## Step 1: Create the "product-admin" Realm Role
1. In the Keycloak admin console, select "pocstore-realm" from the realm dropdown in the top-left
2. Navigate to **Realm roles** in the left sidebar (under "Manage" section)
3. Click the **Create role** button
4. Fill in the role details:
   - **Role name**: `product-admin`
   - **Description**: `Role for product administration access`
5. Click **Save**

## Step 2: Create the "product-admin" Client
1. In the left sidebar, navigate to **Clients** (under "Identity providers" section)
2. Click **Create client**
3. On the "General settings" tab:
   - **Client type**: `OpenID Connect`
   - **Client ID**: `product-admin`
   - **Name**: `Product Admin Service`
   - **Description**: `Client for product administration API access`
4. Click **Next**
5. On the "Capability config" tab:
   - **Client authentication**: `On` (enable)
   - **Standard flow**: `Off`
   - **Direct access grants**: `On` (enable for password-based authentication)
   - **Service accounts roles**: `Off`
6. Click **Next**
7. On the "Login settings" tab:
   - Leave all fields blank (not needed for service-to-service auth)
8. Click **Save**

## Step 3: Configure Client Secret
1. After creating the client, go to the **Credentials** tab of the product-admin client
2. Copy the **Client secret** value (it will be a long UUID-like string)
3. Update your Kubernetes secret file `deploy/k8s/service/product-admin/product-admin-keycloak-secret.yaml`:
   - Replace `PLACEHOLDER_SECRET` with the actual client secret you just copied

## Step 4: Create a User for Testing
1. In the left sidebar, navigate to **Users**
2. Click **Create new user**
3. Fill in user details:
   - **Username**: `product-admin-user` (or your preferred username)
   - **Email**: `admin@pocstore.local` (or any valid email)
   - **First name**: `Product`
   - **Last name**: `Admin`
   - **Email verified**: `On`
4. Click **Create**
5. After creation, go to the **Credentials** tab for this user
6. Click **Set password**
7. Set a password (e.g., `admin123`) and ensure **Temporary** is `Off`
8. Click **Save**

## Step 5: Assign the Role to the User
1. With the user still open, go to the **Role mapping** tab
2. Click **Assign role**
3. In the filter, search for and select the `product-admin` role
4. Click **Assign**

## Step 6: Update Kubernetes Deployment
After updating the client secret in the Kubernetes secret, redeploy your services:

```bash
make services
```

## Verification
Your product-admin service endpoints (under `/api/v1/admin/`) should now be accessible with a Bearer token obtained from Keycloak using this user's credentials.

## Production Notes
Currently your service uses HMAC-SHA256 validation with a hardcoded secret instead of proper RSA validation against Keycloak's JWKS endpoint. For production, you'll want to update the JWT validation in `internal/platform/auth/jwt_validator.go` to use RSA keys from the JWKS URL.