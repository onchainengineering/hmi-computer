#include<stdio.h>
void fun()
{
 static int a=10;
a=a+1;
printf("a=%d\n",a);
}
int main()
{
fun();
fun();
fun();
}
