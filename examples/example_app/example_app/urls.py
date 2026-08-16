"""example_app URL Configuration

The `urlpatterns` list routes URLs to views. For more information please see:
    https://docs.djangoproject.com/en/3.2/topics/http/urls/
Examples:
Function views
    1. Add an import:  from my_app import views
    2. Add a URL to urlpatterns:  path('', views.home, name='home')
Class-based views
    1. Add an import:  from other_app.views import Home
    2. Add a URL to urlpatterns:  path('', Home.as_view(), name='home')
Including another URLconf
    1. Import the include() function: from django.urls import include, path
    2. Add a URL to urlpatterns:  path('blog/', include('blog.urls'))
"""

from django.contrib import admin
from django.urls import include, path
from django.views.generic.base import RedirectView
from django.shortcuts import redirect
from django.contrib.staticfiles.storage import staticfiles_storage

from drf_yasg.views import get_schema_view
from drf_yasg import openapi
from rest_framework import permissions

schema_view = get_schema_view(
    openapi.Info(
        title="Example App API",
        default_version="v0",
        description="This is the API for the example app",
        terms_of_service="Terms of Service goes here",
        contact=openapi.Contact(email="noCRUD_exampleapp@gmailisprofessional.com"),
        license=openapi.License(name="MIT License"),
    ),
    public=True,
    permission_classes=(permissions.AllowAny,),
)

urlpatterns = [
    path(
        "api/docs", schema_view.with_ui("redoc", cache_timeout=0), name="schema-redoc"
    ),
    path("api/", include("api.urls")),
    path("api-auth/", include("rest_framework.urls", namespace="rest_framework")),
    path("api/admin/", admin.site.urls),
    path("", lambda request: redirect("api/", permanent=True)),
    path(
        "favicon.ico",
        RedirectView.as_view(url=staticfiles_storage.url("favicon.ico")),
    ),
]
