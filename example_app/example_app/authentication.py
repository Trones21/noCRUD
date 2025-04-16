"""The Authentication module"""

import re
from multiprocessing import AuthenticationError
from rest_framework import authentication, exceptions
from jwt import InvalidTokenError, PyJWKClient, DecodeError, decode
from django.db.models import CharField, F, Q, Value
from django.contrib.auth.models import User
from example_app import settings
from api.models.users import Users
import secrets
import string


class JWTAccessTokenAuthentication(authentication.BaseAuthentication):
    """JWT auth class"""

    def __init__(self, *args, **kwargs):
        self.use_jwks = settings.USE_JWKS
        if self.use_jwks:
            self.jwks_client = PyJWKClient(settings.OIDC_OP_SIGNATURE_ENDPOINT)
        else:
            with open(settings.LOCAL_PUBLIC_KEY_PATH) as f:
                self.local_public_key = f.read()

    def authenticate(self, request):
        if not settings.USE_JWT_AUTH:
            return (User(), None)

        try:
            header_auth_val = request.META["HTTP_AUTHORIZATION"]
        except Exception as exc:
            raise exceptions.AuthenticationFailed("Authentication Failed") from exc

        match = re.compile(r"^[Bb]earer (.*)$").match(header_auth_val)
        raw_jwt = match.groups()[-1]

        try:
            if self.use_jwks:
                signing_key = self.jwks_client.get_signing_key_from_jwt(raw_jwt)
                data = decode(
                    raw_jwt, signing_key.key, algorithms=["RS256"], audience="viper-vue"
                )
            else:
                data = decode(
                    raw_jwt,
                    self.local_public_key,
                    algorithms=["RS256"],
                    audience="viper-vue",
                )
        except DecodeError as exc:
            raise InvalidTokenError("Token was unable to be decoded") from exc
        except Exception as exc:
            if "Unable to find a signing key" in str(exc):
                raise exceptions.AuthenticationFailed(
                    "JWT does not have a valid Key ID"
                ) from exc

        token_email = data.get("email")
        users = Users.objects.annotate(
            token_email=Value(token_email, CharField())
        ).filter(Q(email__icontains=token_email) | Q(token_email__icontains=F("email")))

        if users.count() > 1:
            raise AuthenticationError("Credentials provided have duplicate")
        elif users.count() < 1:
            raise AuthenticationError("No matching user found.")

        user = users[0]
        authenticated_user = User()
        authenticated_user.username = user.username
        authenticated_user.first_name = user.first_name
        authenticated_user.last_name = user.last_name
        authenticated_user.email = user.email
        authenticated_user.is_active = user.is_active

        return (authenticated_user, raw_jwt)


class PasswordGenerator:
    def generate():
        alphabet = string.ascii_letters + string.digits
        while True:
            password = "".join(secrets.choice(alphabet) for i in range(15))
            if (
                any(c.islower() for c in password)
                and any(c.isupper() for c in password)
                and sum(c.isdigit() for c in password) >= 3
            ):
                return password
