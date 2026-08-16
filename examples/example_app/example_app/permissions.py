import json
from django.http import JsonResponse
from django.contrib.auth.models import Permission
from django.db import transaction
from api.models.users import Users


def get_user_permissions(request):
    data = json.loads(request.body)
    user = Users.objects.get(pk=data["id"])
    if user.is_superuser:
        return JsonResponse({"permissions": list(Permission.objects.all().values())})

    return JsonResponse(
        {"permissions": list(Permission.objects.filter(user__id=user.id).values())}
    )


def get_all_permissions(request):
    permissions = Permission.objects.all()
    permission_list = list(permissions.values())
    return JsonResponse(permission_list, safe=False)


def get_permissions(request):
    try:
        data = json.loads(request.body)
        ids = data["ids"]
        if not isinstance(ids, list):
            raise TypeError()
        permissions = list(Permission.objects.filter(id__in=ids).values())
        return JsonResponse(permissions, safe=False)
    except Exception:
        return JsonResponse(
            {
                "error": 'Please include a body in the following shape {"ids":[<array of ids>]}'
            },
            status=400,
        )


def update_permissions_for_user(user):
    all_permissions = set()
    for role in user.roles.all():
        all_permissions.update(role.permissions.all())
    with transaction.atomic():
        user.user_permissions.clear()
        user.user_permissions.add(*all_permissions)
