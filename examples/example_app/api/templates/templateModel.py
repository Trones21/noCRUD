''' The model, serializer, and viewset for newmodel.
'''

from django.db import models
from django.contrib.auth.decorators import permission_required, login_required
from django.utils.decorators import method_decorator

from rest_framework.serializers import ModelSerializer
from rest_framework.viewsets import ModelViewSet
from django_filters.rest_framework import DjangoFilterBackend
from example_app.authentication import JWTAccessTokenAuthentication

class NewModel(models.Model):
    ''' The template model, not to be imported, just copy and replace NewModel and newmodel with your model name
    '''
    originated_by = models.TextField(default="", blank=True)
    originated_date = models.DateTimeField(auto_now_add=True)
    last_updated_by = models.TextField(default="", blank=True)
    last_updated_date = models.DateTimeField(auto_now=True)

    class Meta:
        '''Settings for the model.
        '''
        managed = True
        db_table = 'newmodel'


class NewModelSerializer(ModelSerializer):
    ''' The serializer.
    '''
    

    class Meta:
        ''' Settings for the serializer.
        '''
        model = NewModel
        fields = '__all__'


class NewModelViewSet(ModelViewSet):
    ''' The viewset for the model.
    '''
    queryset = NewModel.objects.order_by('pk')
    serializer_class = NewModelSerializer
    filter_backends = [DjangoFilterBackend]
    filterset_fields = ['']

    @method_decorator(login_required)
    @method_decorator(permission_required('api.add_newmodel', raise_exception=True))
    def create(self, request):
        return super().create(request)

    def get_queryset(self):
        return super().get_queryset()

    @method_decorator(login_required)
    @method_decorator(permission_required('api.change_newmodel', raise_exception=True))
    def update(self, request, pk=None):
        return super().update(request, pk)

    @method_decorator(login_required)
    @method_decorator(permission_required('api.delete_newmodel', raise_exception=True))
    def destroy(self, request, pk=None):
        return super().destroy(request, pk)